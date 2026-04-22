package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crleonard/pingtower/internal/config"
	"github.com/crleonard/pingtower/internal/httpapi"
	"github.com/crleonard/pingtower/internal/monitor"
	"github.com/crleonard/pingtower/internal/store"
)

func main() {
	cfg := config.Load()

	logger := log.New(os.Stdout, "pingtower ", log.LstdFlags|log.LUTC)

	dataStore, err := store.NewFileStore(cfg.DataFile)
	if err != nil {
		logger.Printf("failed to initialize store error=%v", err)
		os.Exit(1)
	}

	service := monitor.NewService(dataStore, logger, cfg.RequestUserAgent, cfg.MaxHistoryPerCheck)
	service.Start()
	defer service.Stop()

	server := httpapi.NewServer(cfg, logger, dataStore)
	server.SetTriggerer(service)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("pingtower started addr=%s data_file=%s", cfg.ListenAddr, cfg.DataFile)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("http server failed error=%v", err)
			os.Exit(1)
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Printf("graceful shutdown failed error=%v", err)
	}
}
