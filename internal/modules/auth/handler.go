package auth

import (
	"net/http"
	"time"

	"quavixAI/internal/modules/user"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{svc: s}
}

func (h *Handler) Signup(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, token, err := h.svc.Signup(req.Email, req.Password, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour * 24), // Example expiration
		User:      *u,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, token, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour * 24), // Example expiration
		User:      *u,
	})
}

// GetCurrentUser retrieves the current authenticated user's profile.
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// In a real application, you would fetch the user from the database
	// using the userID and return their profile.
	// For now, let's mock a user response.

	// Mock user data (replace with actual database fetch)
	mockUser := user.User{
		ID:    userID.(string),
		Email: "user@example.com", // Replace with actual email from DB
		Name:  "Test User",        // Replace with actual name from DB
		Role:  "user",
	}

	c.JSON(http.StatusOK, mockUser)
}
