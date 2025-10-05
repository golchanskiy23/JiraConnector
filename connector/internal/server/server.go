package server

import (
	"JiraConnector/connector/application"
	"JiraConnector/connector/pb"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type ImplementedJiraService struct {
	pb.UnimplementedJiraPullerServer
}

func (impl *ImplementedJiraService) mustEmbedUnimplementedJiraPullerServer() {}

func (impl *ImplementedJiraService) GetProjects(ctx context.Context, request *pb.ProjectsRequest) (*pb.ProjectsResponse, error) {
	params, err := validateParams(request)
	if err != nil {
		return nil, err
	}
	projects, err := Get(params)
	if err != nil {
		return nil, err
	}
	return projects, nil
}
func (impl *ImplementedJiraService) FetchProject(context.Context, *pb.Key) (*pb.ID, error) {
	//Fetch()
	return &pb.ID{Id: uint32(5)}, nil
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

func Get(params *pb.ProjectsRequest) (*pb.ProjectsResponse, error) {
	response, err := GetJiraResponse(jiraUrlAllProjects()) // *http.Response
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(response.Body)
	defer response.Body.Close()

	var projects []Project
	json.Unmarshal(body, &projects)
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
	ID          uint
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
