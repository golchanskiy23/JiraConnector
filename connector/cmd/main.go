package main

import (
	"JiraConnector/connector/application"
	"JiraConnector/connector/internal/server"
	"JiraConnector/connector/internal/util"
	"JiraConnector/connector/pb"
	"net"
	"os"

	"google.golang.org/grpc"
)

func main() {
	log, err := util.SetupLogger()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	if err := application.Applicate(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	log.Info("Successfull configuration pull")

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
