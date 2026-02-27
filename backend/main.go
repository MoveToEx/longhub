package main

import (
	"context"
	"log"
	"long/internal/config"
	"long/internal/db"
	"long/internal/middleware"
	"long/internal/queue"
	"long/internal/routes"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	config.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn, err := pgxpool.New(ctx, config.GetConfig().DatabaseURL)

	if err != nil {
		log.Fatalln("Failed when connecting to database: ", err)
		return
	}

	if err := db.Migrate(ctx, stdlib.OpenDBFromPool(conn)); err != nil {
		log.Fatalln("Failed when applying migrations: ", err)
		return
	}

	db.Init(ctx, conn)
	queue.Init(conn)
	config.InitS3()

	if err := config.InitWebAuthn(); err != nil {
		log.Fatalln("Failed when initializing WebAuthn. Check your environment variables.")
		return
	}

	app := gin.New()

	app.Use(gin.Recovery(), middleware.CORSMiddleware(config.GetConfig().CORSOrigin))

	routes.RegisterRoutes(app)

	srv := &http.Server{
		Addr:              ":8000",
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Unable to serve: ", err)
		}
	}()

	<-ctx.Done()

	stop()
	log.Println("Shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := queue.Stop(); err != nil {
		log.Println("Cannot stop queue: ", err)
	}

	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Server forced to shutdown: ", err)
	}
}
