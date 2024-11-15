package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"logger-service/data"
	"logger-service/logs"
	"net"
)

type LogServer struct {
	logs.UnimplementedLogServiceServer
	Models data.Models
}

func (l *LogServer) WriteLog(ctx context.Context, req *logs.LogRequest) (*logs.LogResponse, error) {
	input := req.GetLogEntry()

	logEntry := data.LogEntry{
		Name: input.Name,
		Data: input.Data,
	}

	err := l.Models.LogEntry.Insert(logEntry)
	if err != nil {
		res := &logs.LogResponse{Result: "failed"}
		return res, err
	}

	res := &logs.LogResponse{
		Result: "logged",
	}
	return res, nil
}

func (app *Config) gRPCListen() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen for grpc: %v", err)
	}

	s := grpc.NewServer()
	logs.RegisterLogServiceServer(s, &LogServer{Models: data.Models{}})

	log.Printf("gRPC server listening on port %s", grpcPort)

	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to listen for grpc: %v", err)
	}
}
