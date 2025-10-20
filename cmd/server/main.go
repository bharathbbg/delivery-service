package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bharathbbg/delivery-service/internal/api/grpc"
	"github.com/bharathbbg/delivery-service/internal/api/rest"
	"github.com/bharathbbg/delivery-service/internal/config"
	"github.com/bharathbbg/delivery-service/internal/repository"
	"github.com/bharathbbg/delivery-service/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize repository
	repo, err := repository.NewPostgresRepository(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// Initialize cache
	cache, err := repository.NewRedisCache(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
	defer cache.Close()

	// Initialize service
	svc := service.NewDeliveryService(repo, cache)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(svc)
	go func() {
		lis, err := net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", cfg.GRPCAddr, err)
		}
		log.Printf("gRPC server listening on %s", cfg.GRPCAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Initialize REST server
	router := rest.NewRouter(svc)
	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	// Start HTTP server
	go func() {
		log.Printf("REST server listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	grpcServer.GracefulStop()
	log.Println("Server exited properly")
}