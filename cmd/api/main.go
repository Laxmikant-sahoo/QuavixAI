package main

import (
	"context"
	"log"

	"quavixAI/internal/config"
	"quavixAI/internal/db"

	// modules
	authModule "quavixAI/internal/modules/auth"
	chatModule "quavixAI/internal/modules/chat"
	llmModule "quavixAI/internal/modules/llm"
	vectorModule "quavixAI/internal/modules/vector"

	// middleware
	"quavixAI/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// ==============================
	// Load config
	// ==============================
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	// ==============================
	// DB
	// ==============================
	pgClient, err := db.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres error: %v", err)
	}
	pg := pgClient.DB

	// Redis
	rdsClient, err := db.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis error: %v", err)
	}

	// ==============================
	// Vector Store (pgvector)
	// ==============================
	vectorStore := vectorModule.NewPgVectorStore(pg, 384) // 384 = embedding dim from schema
	if err := vectorStore.Init(context.Background()); err != nil {
		log.Fatalf("vector init error: %v", err)
	}

	// ==============================
	// LLM Manager
	// ==============================
	llmManager, err := llmModule.NewManager(llmModule.ManagerConfig{
		Provider: "local",  // ollama/local
		Model:    "llama3", // example
	})
	if err != nil {
		log.Fatalf("llm error: %v", err)
	}

	// ==============================
	// Repositories
	// ==============================
	authRepo := authModule.NewRepository(pg)
	chatRepo := chatModule.NewRepository(pg)

	// ==============================
	// Services
	// ==============================
	jwtSvc := authModule.NewJWT(cfg.JWTSecret) // <-- real JWT constructor
	authService := authModule.NewService(authRepo, jwtSvc)

	memoryEngine := chatModule.NewMemoryEngine(rdsClient, vectorStore, llmManager)

	chatService := chatModule.NewService(chatModule.ServiceConfig{
		Repo:      chatRepo,
		LLM:       llmManager,
		Vector:    vectorStore,
		Memory:    memoryEngine,
		FiveWhy:   true,
		Evaluator: true,
		RootCause: true,
		Reframer:  true,
	})

	// ==============================
	// Handlers
	// ==============================
	authHandler := authModule.NewHandler(authService)
	chatHandler := chatModule.NewHandler(chatService)

	// ==============================
	// Gin Router
	// ==============================
	r := gin.Default()

	// ==============================
	// Routes
	// ==============================
	api := r.Group("/api/v1")

	// Auth
	api.POST("/auth/signup", authHandler.Signup)
	api.POST("/auth/login", authHandler.Login)

	// Protected
	protected := api.Group("")
	protected.Use(middleware.JWTAuthMiddleware(cfg))

	// User
	protected.GET("/me", authHandler.GetCurrentUser)

	// Chat / AI
	protected.POST("/chat", chatHandler.Chat)
	protected.POST("/chat/5why", chatHandler.FiveWhy)
	protected.POST("/chat/root-cause", chatHandler.RootCause)
	protected.POST("/chat/reframe", chatHandler.Reframe)
	protected.POST("/chat/memory/compress", chatHandler.CompressSession)
	protected.POST("/chat/memory/recall", chatHandler.Recall)

	// ==============================
	// Start server
	// ==============================
	log.Println("API running on :" + cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
