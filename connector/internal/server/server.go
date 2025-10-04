package server

import (
	"JiraConnector/connector/pb"
	"context"
)

type ImplementedJiraService struct {
	pb.UnimplementedJiraPullerServer
}

func (impl *ImplementedJiraService) mustEmbedUnimplementedJiraPullerServer() {}

func (impl *ImplementedJiraService) GetProjects(context.Context, *pb.ProjectsRequest) (*pb.ProjectsResponse, error) {
	return nil, nil
}
func (impl *ImplementedJiraService) GetProject(context.Context, *pb.Key) (*pb.ID, error) {
	return &pb.ID{Id: uint32(5)}, nil
}
