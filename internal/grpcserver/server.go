package grpcserver

import (
	"context"
	"net"

	"brook/internal/middleware"

	"github.com/Chandra179/gosdk/logger"
	"google.golang.org/grpc"
)

func Server() {
	log := logger.NewLogger("dev")

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RequestIDUnaryInterceptor,
		),
	)

	// Register your gRPC service implementations here, e.g.:
	// pb.RegisterOrdersServer(srv, &ordersHandler{})

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Error(context.Background(), "grpc listen error", logger.Field{Key: "error", Value: err.Error()})
		return
	}
	log.Info(context.Background(), "grpc server starting", logger.Field{Key: "addr", Value: lis.Addr().String()})
	if err := srv.Serve(lis); err != nil {
		log.Error(context.Background(), "grpc server error", logger.Field{Key: "error", Value: err.Error()})
	}
}
