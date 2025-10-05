package server

import (
	"JiraConnector/connector/internal/entity"
	"JiraConnector/connector/internal/usecase"
	"JiraConnector/connector/internal/util"
	"JiraConnector/connector/pb"
	"context"
	"net/http"
	"strconv"
)

type ImplementedJiraService struct {
	pb.UnimplementedJiraPullerServer
}

func (impl *ImplementedJiraService) mustEmbedUnimplementedJiraPullerServer() {}

func (impl *ImplementedJiraService) GetProjects(ctx context.Context, request *pb.ProjectsRequest) (*pb.ProjectsResponse, error) {
	params, err := util.ValidateParams(request)
	if err != nil {
		return nil, err
	}
	projects, err := usecase.Get(ctx, &http.Client{}, params)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (impl *ImplementedJiraService) FetchProject(ctx context.Context, k *pb.Key) (*pb.ID, error) {
	if err := usecase.Fetch(ctx, k.Key); err != nil {
		return nil, err
	}

	project := new(entity.Project)
	if err := util.GetJSON(ctx, &http.Client{}, usecase.JiraUrlProjectWithKey(k.Key), project); err != nil {
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
