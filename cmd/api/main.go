package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quavixAI/internal/config"
	"quavixAI/internal/db"
	"quavixAI/internal/router"
	"quavixAI/internal/server"

	// modules
	authModule "quavixAI/internal/modules/auth"
	chatModule "quavixAI/internal/modules/chat"
	llmModule "quavixAI/internal/modules/llm"
	userModule "quavixAI/internal/modules/user"
	vectorModule "quavixAI/internal/modules/vector"

	// middlewares
	"quavixAI/internal/middleware"

	// utils
	"quavixAI/pkg/logger"
)

func main() {
	// ==============================
	// Context + graceful shutdown
	// ==============================
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// ==============================
	// Load config
	// ==============================
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	// ==============================
	// Logger
	// ==============================
	appLogger := logger.New(cfg.App.Env)
	appLogger.Info("QuavixAI API starting...")

	// ==============================
	// Databases
	// ==============================
	// PostgreSQL
	pgClient, err := db.NewPostgres(cfg.Database.PostgresURL)
	if err != nil {
		appLogger.Fatal("postgres connection failed", err)
	}
	defer pgClient.Close()

	pg := pgClient.DB

	// Redis (cache + vector memory + session)
	rdsClient, err := db.NewRedis(cfg.Database.RedisURL)
	if err != nil {
		appLogger.Fatal("redis connection failed", err)
	}
	defer rdsClient.Close()

	rds := rdsClient.Client

	// ==============================
	// Vector DB (pgvector / faiss / hybrid)
	// ==============================
	vectorStore, err := vectorModule.NewStore(vectorModule.StoreConfig{
		Type:      cfg.Vector.Type, // pgvector | faiss | redis
		Postgres:  pg,
		Redis:     rds,
		Dimension: cfg.Vector.Dimension,
	})
	if err != nil {
		appLogger.Fatal("vector store init failed", err)
	}

	// ==============================
	// LLM Engine
	// ==============================
	llmManager, err := llmModule.NewManager(llmModule.ManagerConfig{
		Provider:  cfg.LLM.Provider, // openai | local | ollama | custom
		APIKey:    cfg.LLM.APIKey,
		Model:     cfg.LLM.Model,
		Embedding: cfg.LLM.EmbeddingModel,
		Vector:    vectorStore,
		Redis:     rds,
		Postgres:  pg,
		FiveWhy:   true, // enable 5-why reasoning mode
		RootCause: true,
	})
	if err != nil {
		appLogger.Fatal("llm manager init failed", err)
	}

	// ==============================
	// Repositories
	// ==============================
	userRepo := userModule.NewRepository(pg)
	authRepo := authModule.NewRepository(pg)
	chatRepo := chatModule.NewRepository(pg)

	// ==============================
	// Services
	// ==============================
	userService := userModule.NewService(userRepo)
	authService := authModule.NewService(authRepo, cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry)

	chatService := chatModule.NewService(chatModule.ServiceConfig{
		Repo:      chatRepo,
		LLM:       llmManager,
		Vector:    vectorStore,
		Redis:     rds,
		FiveWhy:   true,
		Evaluator: true,
		RootCause: true,
		Reframer:  true,
	})

	// ==============================
	// Handlers
	// ==============================
	userHandler := userModule.NewHandler(userService)
	authHandler := authModule.NewHandler(authService)
	chatHandler := chatModule.NewHandler(chatService)

	// ==============================
	// Router
	// ==============================
	r := router.New()

	// Global middleware
	r.Use(middleware.Logging(appLogger))
	r.Use(middleware.CORS(cfg.App.AllowedOrigins))
	r.Use(middleware.RateLimit(cfg.App.RateLimit))

	// Health
	r.GET("/health", func(c router.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// ==============================
	// API Routes
	// ==============================
	api := r.Group("/api/v1")

	// Auth
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)

	// Protected
	protected := api.Group("")
	protected.Use(middleware.JWT(cfg.Auth.JWTSecret))

	// Users
	protected.GET("/users/me", userHandler.Me)

	// Chat / 5-Why Engine
	protected.POST("/chat", chatHandler.Chat)
	protected.POST("/chat/5why", chatHandler.FiveWhy)
	protected.POST("/chat/root-cause", chatHandler.RootCause)
	protected.POST("/chat/reframe", chatHandler.Reframe)

	// ==============================
	// Server
	// ==============================
	srv := server.New(server.Config{
		Address:      cfg.App.Address,
		Port:         cfg.App.Port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, r)

	go func() {
		if err := srv.Start(); err != nil {
			appLogger.Fatal("server start failed", err)
		}
	}()

	appLogger.Info("API running on ", cfg.App.Address, ":", cfg.App.Port)

	// ==============================
	// Graceful shutdown
	// ==============================
	<-stop
	appLogger.Info("Shutting down API...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("server shutdown error", err)
	}

	appLogger.Info("Shutdown complete")
}
