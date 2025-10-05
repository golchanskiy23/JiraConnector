package usecase

import (
	"JiraConnector/connector/application"
	"JiraConnector/connector/internal/dto"
	"JiraConnector/connector/internal/entity"
	"JiraConnector/connector/internal/util"
	"JiraConnector/connector/pb"
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func Get(ctx context.Context, client *http.Client, params *pb.ProjectsRequest) (*pb.ProjectsResponse, error) {
	var projects []entity.Project
	if err := util.GetJSON(ctx, client, JiraUrlAllProjects(), params); err != nil {
		return nil, err
	}

	pageInfo := &dto.PageInfo{}
	projects = util.Filter(projects, func(project entity.Project) bool {
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

func Fetch(ctx context.Context, key string) error {
	project := new(entity.Project)
	if err := util.GetJSON(ctx, &http.Client{}, JiraUrlProjectWithKey(key), project); err != nil {
		return err
	}

	issues, err := fetchIssues(ctx, &http.Client{}, project)
	if err != nil {
		return err
	}
	issuesT := entity.Issues{
		MaxResults: len(issues),
		Data:       issues,
	}
	fmt.Println(issuesT.MaxResults)
	for i := 0; i < len(issuesT.Data); i++ {
		fmt.Println(issuesT.Data[0].Key)
	}
	// insert Issues to DB
	return nil
}

func fetchIssues(ctx context.Context, client *http.Client, project *entity.Project) ([]entity.Issue, error) {
	cfg := application.App.Config()
	perPage := int(cfg.ServiceConfig.IssueInOneReq)
	if perPage <= 0 {
		perPage = 50
	}

	var info entity.IssuesInfo
	if err := util.GetJSON(ctx, client, JiraUrlIssuesInfo(project.Key), &info); err != nil {
		return nil, err
	}
	total := info.Total
	if total == 0 {
		return nil, nil
	}

	pages := (total + perPage - 1) / perPage
	eg, ctx := errgroup.WithContext(ctx)
	sem := semaphore.NewWeighted(int64(cfg.ServiceConfig.Thread))
	issuesCh := make(chan []entity.Issue, pages)

	for p := range pages {
		start := p * perPage
		eg.Go(func() error {
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			var chunk entity.Issues
			url := JiraUrlIssues(project.Key, start)
			if err := util.GetJSON(ctx, client, url, &chunk); err != nil {
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

	issues := make([]entity.Issue, 0, total)
	for chunk := range issuesCh {
		issues = append(issues, chunk...)
	}

	if err := <-errCh; err != nil {
		return nil, err
	}
	return issues, nil
}

func JiraUrlAllProjects() string {
	cfg := application.App.Config()
	return fmt.Sprintf("%s%s", cfg.ServiceConfig.JiraURL, "project")
}

func JiraUrlProjectWithKey(key string) string {
	cfg := application.App.Config()
	return fmt.Sprintf("%s%s%s", cfg.ServiceConfig.JiraURL, "project/", key)
}

func JiraUrlIssuesInfo(name string) string {
	cfg := application.App.Config()
	return fmt.Sprintf("%ssearch?jql=project=%s", cfg.ServiceConfig.JiraURL, name)
}

func JiraUrlIssues(name string, startedAt int) string {
	cfg := application.App.Config()
	return fmt.Sprintf("%ssearch?jql=project=%s&expand=changelog&startAt=%d&maxResults=%d",
		cfg.ServiceConfig.JiraURL,
		name,
		startedAt,
		int(cfg.ServiceConfig.IssueInOneReq),
	)
}
