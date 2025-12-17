package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChinawatDc/go-api-grpc-gateway/internal/transport/gateway"
)

func main() {
	cfg := gateway.Config{
		HTTPAddr: ":8080",
		GRPCAddr: "localhost:50051",
	}

	srv, err := gateway.NewHTTPServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Println("HTTP gateway listening on", cfg.HTTPAddr, "-> gRPC", cfg.GRPCAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("HTTP gateway stopped")
}
