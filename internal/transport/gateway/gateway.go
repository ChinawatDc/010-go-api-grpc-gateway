package gateway

import (
	"context"
	"net/http"
	"time"

	userv1 "github.com/ChinawatDc/go-api-grpc-gateway/gen/go/user/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Config struct {
	HTTPAddr string
	GRPCAddr string
}

func NewHTTPServer(cfg Config) (*http.Server, error) {
	mux := runtime.NewServeMux()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := []grpc.DialOption{grpc.WithInsecure()}

	if err := userv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, cfg.GRPCAddr, opts); err != nil {
		return nil, err
	}

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return srv, nil
}