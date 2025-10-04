package main

import (
	"JiraConnector/connector/config"
	"JiraConnector/connector/internal/server"
	"JiraConnector/connector/internal/util"
	"JiraConnector/connector/pb"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.NewConfig()
	log, err := util.SetupLogger()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	log.Info("Successfull configuration pull")
	log.Info(fmt.Sprintf("Configuration internals: %v", cfg))

	lis, err := net.Listen("tcp", ":8567")
	if err != nil {
		log.Error("Error during get tcp connection with server")
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterJiraPullerServer(grpcServer, &server.ImplementedJiraService{})
	if err := grpcServer.Serve(lis); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
