package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/RajVerma97/golang-vercel/backend/internal/config"
)

type HTTPServer struct {
	Server *http.Server
}

func NewHTTPServer(config *config.ServerConfig) (*HTTPServer, error) {
	serverAddr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	server := &HTTPServer{
		Server: &http.Server{
			Addr: serverAddr,
		},
	}
	return server, nil
}

func (s *HTTPServer) Start(ctx context.Context) error {
	// Start server in background
	go func() {
		if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()
	fmt.Printf("\nSUCCESSFULLY STARTED SERVER %s\n", s.Server.Addr)

	// Listen for shutdown signal
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		_ = s.Stop(shutdownCtx)
	}()

	return nil
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	err := s.Server.Shutdown(ctx)
	if err != nil {
		fmt.Println("FAILED TO STOP SERVER")
		return err
	}
	fmt.Println("stopped SERVER")
	return nil
}
