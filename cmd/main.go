package main

import (
	"context"
	"errors"
	"github.com/joho/godotenv"
	"github.com/onemgvv/storage-service.git/internal/config"
	deliveryHttp "github.com/onemgvv/storage-service.git/internal/delivery/http"
	"github.com/onemgvv/storage-service.git/internal/repository"
	"github.com/onemgvv/storage-service.git/internal/server"
	"github.com/onemgvv/storage-service.git/internal/service"
	"github.com/onemgvv/storage-service.git/pkg/database"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const configDir = "configs"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("[ENV Load] || [Failed]: %s", err.Error())
	}

	cfg, err := config.Init(configDir)
	if err != nil {
		log.Fatalf("[Config Load] || [Failed]: %s", err.Error())
	}

	db, err := database.Init(cfg)
	if err != nil {
		log.Fatalf("[Database INIT] || [Failed]: %s", err.Error())
	}

	repositories := repository.NewRepositories(db)
	services := service.NewServices(&service.Deps{
		Repos: repositories,
	})
	handlers := deliveryHttp.NewHandler(services)

	app := server.NewServer(cfg, handlers.Init(cfg))

	go func() {
		if err = app.Run(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[SERVER START] || [FAILED]: %s", err.Error())
		}
	}()

	log.Println("Application started")

	/**
	 *	Graceful Shutdown
	 */
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	const timeout = 5 * time.Second
	ctx, shutdown := context.WithTimeout(context.Background(), timeout)
	defer shutdown()

	if err = app.Shutdown(ctx); err != nil {
		log.Fatalf("[SERVER STOP] || [FAILED]: %s", err.Error())
	}

	if err = database.Close(db); err != nil {
		log.Fatalf("[DATABASE CONN CLOSE] || [FAILED]: %s", err.Error())
	}
}
