package server

import (
	"JiraConnector/connector/application"
	"JiraConnector/connector/pb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type ImplementedJiraService struct {
	pb.UnimplementedJiraPullerServer
}

func (impl *ImplementedJiraService) mustEmbedUnimplementedJiraPullerServer() {}

func getJSON(ctx context.Context, client *http.Client, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		limited := io.LimitReader(response.Body, 2048)
		body, _ := io.ReadAll(limited)
		return fmt.Errorf("request %s returned status %d: %s", url, response.StatusCode, string(body))
	}

	return json.NewDecoder(response.Body).Decode(target)
}

func (impl *ImplementedJiraService) GetProjects(ctx context.Context, request *pb.ProjectsRequest) (*pb.ProjectsResponse, error) {
	params, err := validateParams(request)
	if err != nil {
		return nil, err
	}
	projects, err := Get(ctx, &http.Client{}, params)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func Get(ctx context.Context, client *http.Client, params *pb.ProjectsRequest) (*pb.ProjectsResponse, error) {
	var projects []Project
	if err := getJSON(ctx, client, jiraUrlAllProjects(), params); err != nil {
		return nil, err
	}

	pageInfo := &PageInfo{}
	projects = filter(projects, func(project Project) bool {
		return strings.HasPrefix(strings.ToLower(project.Name), strings.ToLower(params.Search)) ||
			strings.HasPrefix(strings.ToLower(project.Key), strings.ToLower(params.Search))
	})

	pageInfo.PageCount = int32(len(projects)) / params.Limit
	if int32(len(projects))%params.Limit != 0 {
		pageInfo.PageCount++
	}
	pageInfo.ProjectsCount = int32(len(projects))

	if params.Page*params.Limit < int32(len(projects)) {
		projects = projects[(params.Page-1)*params.Limit : params.Page*params.Limit]
		pageInfo.CurrentPage = params.Page
	} else {
		projects = projects[(params.Page-1)*params.Limit:]
		pageInfo.CurrentPage = 1
	}

	result := &pb.ProjectsResponse{
		Projects: make([]*pb.Project, len(projects)),
		PageInfo: &pb.PageInfo{
			PageCount:     int32(pageInfo.PageCount),
			CurrentPage:   int32(pageInfo.CurrentPage),
			ProjectsCount: int32(pageInfo.ProjectsCount),
		},
	}

	for i, project := range projects {
		result.Projects[i] = &pb.Project{
			Key:         project.Key,
			Name:        project.Name,
			Url:         project.URL,
			Description: project.Description,
		}
	}

	return result, nil
}

func (impl *ImplementedJiraService) FetchProject(ctx context.Context, k *pb.Key) (*pb.ID, error) {
	/*id, err := Fetch(ctx, k.Key)
	if err != nil {
		return nil, err
	}
	return &pb.ID{Id: uint32(id)}, nil*/
	project := new(Project)
	if err := getJSON(ctx, &http.Client{}, jiraUrlProjectWithKey(k.Key), project); err != nil {
		return nil, err
	}

	ans, err := strconv.Atoi(project.ID)
	if err != nil {
		return nil, err
	}

	return &pb.ID{
		Id: uint32(ans),
	}, nil
}

func Fetch(ctx context.Context, key string) (int, error) {
	project := new(Project)
	if err := getJSON(ctx, &http.Client{}, jiraUrlProjectWithKey(key), project); err != nil {
		return math.MaxInt, err
	}

	issues, err := fetchIssues(ctx, &http.Client{}, project)
	if err != nil {
		return math.MaxInt, err
	}
	issuesT := Issues{
		MaxResults: len(issues),
		Data:       issues,
	}
	fmt.Println(issuesT.MaxResults)
	for i := 0; i < len(issuesT.Data); i++ {
		fmt.Println(issuesT.Data[0].Key)
	}
	return issuesT.MaxResults, nil
}

func fetchIssues(ctx context.Context, client *http.Client, project *Project) ([]Issue, error) {
	cfg := application.App.Config()
	perPage := int(cfg.ServiceConfig.IssueInOneReq)
	if perPage <= 0 {
		perPage = 50
	}

	var info IssuesInfo
	if err := getJSON(ctx, client, jiraUrlIssuesInfo(project.Key), &info); err != nil {
		return nil, err
	}
	total := info.Total
	if total == 0 {
		return nil, nil
	}

	pages := (total + perPage - 1) / perPage
	eg, ctx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(cfg.ServiceConfig.Thread))
	issuesCh := make(chan []Issue, pages)

	for p := range pages {
		start := p * perPage
		eg.Go(func() error {
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			var chunk Issues
			url := jiraUrlIssues(project.Key, start)
			if err := getJSON(ctx, client, url, &chunk); err != nil {
				return err
			}
			if len(chunk.Data) > 0 {
				issuesCh <- chunk.Data
			}
			return nil
		})
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- eg.Wait()
		close(issuesCh)
		close(errCh)
	}()

	issues := make([]Issue, 0, total)
	for chunk := range issuesCh {
		issues = append(issues, chunk...)
	}

	if err := <-errCh; err != nil {
		return nil, err
	}
	return issues, nil
}

func validateParams(request *pb.ProjectsRequest) (*pb.ProjectsRequest, error) {
	limit, err := validate(request.Limit, func(int32) bool {
		if request.Limit < 0 {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	page, err := validate(request.Page, func(int32) bool {
		if request.Page < 0 {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return &pb.ProjectsRequest{
		Limit:  limit,
		Page:   page,
		Search: request.Search,
	}, nil
}

func validate[T any](input T, f func(T) bool) (T, error) {
	if !f(input) {
		var zero T
		return zero, fmt.Errorf("invalid value in response: %v", input)
	}
	return input, nil
}

func filter[T any](a []T, f func(T) bool) (b []T) {
	for _, c := range a {
		if f(c) {
			b = append(b, c)
		}
	}
	return
}

func jitterBackoff(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	delta := max - min
	return min + time.Duration(rand.Int63n(int64(delta)))
}

func GetJiraResponse(url string) (*http.Response, error) {
	cfg := application.App.Config()

	minWait := *cfg.ServiceConfig.MinTimeSleep
	maxWait := *cfg.ServiceConfig.MaxTimeSleep

	waitTime := minWait
	for {
		resp, err := http.Get(url)
		if err == nil {
			return resp, nil
		}

		if waitTime > maxWait {
			return nil, errors.New("waiting limit exceeded")
		}

		delay := jitterBackoff(waitTime/2, waitTime)
		fmt.Printf("Request failed: %v — retrying in %v\n", err, delay)
		time.Sleep(delay)
		waitTime *= 2
	}
}

type Project struct {
	ID          string
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"self"`
	Description string `json:"description"`
}

type IssuesInfo struct {
	StartAt    int `json:"startAt"`
	MaxResults int `json:"maxResults"`
	Total      int `json:"total"`
}

type Issues struct {
	MaxResults int     `json:"maxResults"`
	Data       []Issue `json:"issues"`
}

type Issue struct {
	Key       string      `json:"key"`
	Fields    IssueFields `json:"fields"`
	ChangeLog ChangeLog   `json:"changelog"`
}

type IssueFields struct {
	CreatedTime string        `json:"created"`
	UpdatedTime string        `json:"updated"`
	Description string        `json:"description"`
	Summary     string        `json:"summary"`
	Creator     Author        `json:"creator"`
	Assignee    Author        `json:"reporter"`
	TimeSpent   int           `json:"timespent"`
	Type        IssueType     `json:"issuetype"`
	Status      IssueStatus   `json:"status"`
	Priority    IssuePriority `json:"priority"`
}

type Author struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type IssueType struct {
	Name string `json:"name"`
}

type IssueStatus struct {
	Name string `json:"name"`
}

type IssuePriority struct {
	Name string `json:"name"`
}

type ChangeLog struct {
	Count     int       `json:"total"`
	Histories []History `json:"histories"`
}

type History struct {
	Author      Author         `json:"author"`
	CreatedTime string         `json:"created"`
	Items       []StatusChange `json:"items"`
}

type StatusChange struct {
	FromStatus string `json:"fromString"`
	ToStatus   string `json:"toString"`
}

type PageInfo struct {
	PageCount     int32 `json:"pageCount"`
	CurrentPage   int32 `json:"currentPage"`
	ProjectsCount int32 `json:"projectsCount"`
}

func jiraUrlAllProjects() string {
	cfg := application.App.Config()
	return fmt.Sprintf("%s%s", cfg.ServiceConfig.JiraURL, "project")
}

func jiraUrlProjectWithKey(key string) string {
	cfg := application.App.Config()
	return fmt.Sprintf("%s%s%s", cfg.ServiceConfig.JiraURL, "project/", key)
}

func jiraUrlIssuesInfo(name string) string {
	cfg := application.App.Config()
	return fmt.Sprintf("%ssearch?jql=project=%s", cfg.ServiceConfig.JiraURL, name)
}

func jiraUrlIssues(name string, startedAt int) string {
	cfg := application.App.Config()
	return fmt.Sprintf("%ssearch?jql=project=%s&expand=changelog&startAt=%d&maxResults=%d",
		cfg.ServiceConfig.JiraURL,
		name,
		startedAt,
		int(cfg.ServiceConfig.IssueInOneReq),
	)
}
