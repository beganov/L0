package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beganov/L0/internal/api"
	"github.com/beganov/L0/internal/broker"
	"github.com/beganov/L0/internal/cache"
	"github.com/beganov/L0/internal/config"
	"github.com/beganov/L0/internal/database"
	"github.com/beganov/L0/internal/logger"
	"github.com/beganov/L0/internal/metrics"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loadEnvOrExit() // dotenv + config
	metrics.Init()  // Prometheus
	config.VarsInit()

	handleSignals(cancel) // Ctrl+C/SIGTERM

	database.RunMigrations(config.PostgresURL)

	db := database.InitDB(ctx, config.PostgresURL)
	defer db.Close()

	reader := initKafkaReader()
	defer reader.Close()

	orderCache := initCache(ctx, db)

	httpSrv := startHTTPServer(orderCache, db)

	// Kafka consumer
	go broker.ConsumeKafka(ctx, reader, db, orderCache)

	<-ctx.Done()
	logger.Info("Shutting down services")

	gracefulShutdown(httpSrv, db, reader)
	logger.Info("App stopped")
}

// Load environment variables or exit
func loadEnvOrExit() {
	if err := godotenv.Load(); err != nil {
		logger.Fatal(err, "No .env file found")
	}
}

// Handle OS signals
func handleSignals(cancel context.CancelFunc) {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		sig := <-c
		logger.Info("Caught signal", sig)
		cancel()
	}()
}

// Init Kafka consumer with basic logging
func initKafkaReader() *kafka.Reader {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{config.KafkaBroker},
		Topic:   config.KafkaTopic,
		GroupID: config.KafkaGroupID,
	})
	logger.Info("Kafka reader initialized for topic", config.KafkaTopic)
	return r
}

// Init cache and try restoring from DB
func initCache(ctx context.Context, db *pgxpool.Pool) *cache.OrderCache {
	c := cache.NewOrderCache(config.CacheCap)
	if err := database.LoadCacheFromDB(ctx, db, c); err != nil {
		metrics.DBErrorsTotal.Inc()
		logger.Error(err, "Failed to fully restore cache")
	} else {
		logger.Info("Cache restored from DB")
	}
	return c
}

// Start HTTP server
func startHTTPServer(orderCache *cache.OrderCache, db *pgxpool.Pool) *http.Server {
	srv := &http.Server{
		Addr:    config.HttpAddr,
		Handler: api.SetupRouter(orderCache, db),
	}
	go func() {
		logger.Info("HTTP server running at", config.HttpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(err, "HTTP server failed")
		}
	}()
	return srv
}

// Graceful shutdown for all services
func gracefulShutdown(srv *http.Server, db *pgxpool.Pool, reader *kafka.Reader) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(err, "HTTP server shutdown failed")
	}

	db.Close()

	if err := reader.Close(); err != nil {
		logger.Error(err, "Kafka reader shutdown failed")
	}

	logger.Info("All services stopped")
}
