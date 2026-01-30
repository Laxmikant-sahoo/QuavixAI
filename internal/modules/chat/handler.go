package chat

import (
	"net/http"
	"time"

	"quavixAI/pkg/response"
)

// ================================
// Handler
// ================================

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// ================================
// Request DTOs
// ================================

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type FiveWhyRequest struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
}

type RootCauseRequest struct {
	Steps []FiveWhyStep `json:"steps"`
}

type ReframeRequest struct {
	Question string          `json:"question"`
	Root     RootCauseResult `json:"root_cause"`
}

type CompressRequest struct {
	SessionID string `json:"session_id"`
}

type RecallRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// ================================
// Core Chat Endpoint
// ================================

func (h *Handler) Chat(c response.Context) error {
	var req ChatRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("invalid request body"))
	}

	userID := c.GetString("user_id")

	resp, err := h.service.Chat(c.Context(), req.SessionID, userID, req.Message)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Success(map[string]interface{}{
		"reply":      resp.Text,
		"confidence": resp.Confidence,
		"latency_ms": resp.Latency.Milliseconds(),
	}))
}

// ================================
// 5-Why Endpoint
// ================================

func (h *Handler) FiveWhy(c response.Context) error {
	var req FiveWhyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("invalid request body"))
	}

	userID := c.GetString("user_id")

	session, err := h.service.FiveWhy(c.Context(), req.SessionID, userID, req.Question)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Success(session))
}

// ================================
// Root Cause Endpoint
// ================================

func (h *Handler) RootCause(c response.Context) error {
	var req RootCauseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("invalid request body"))
	}

	rc, err := h.service.RootCause(c.Context(), req.Steps)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Success(rc))
}

// ================================
// Reframe Endpoint
// ================================

func (h *Handler) Reframe(c response.Context) error {
	var req ReframeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("invalid request body"))
	}

	ref, err := h.service.Reframe(c.Context(), req.Question, req.Root)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Success(ref))
}

// ================================
// Memory Endpoints
// ================================

func (h *Handler) CompressSession(c response.Context) error {
	var req CompressRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("invalid request body"))
	}

	summary, err := h.service.CompressSession(c.Context(), req.SessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Success(map[string]interface{}{
		"summary": summary,
		"ts":      time.Now(),
	}))
}

func (h *Handler) Recall(c response.Context) error {
	var req RecallRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("invalid request body"))
	}

	mem, err := h.service.Recall(c.Context(), req.Query, req.Limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
	}

	return c.JSON(http.StatusOK, response.Success(mem))
}
