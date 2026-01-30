package response

import (
	"context"
	"encoding/json"
	"net/http"
)

// ================================
// Context Wrapper
// ================================

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	values  map[string]interface{}
}

func NewContext(w http.ResponseWriter, r *http.Request) Context {
	return Context{
		Writer:  w,
		Request: r,
		values:  make(map[string]interface{}),
	}
}

// ================================
// Helpers
// ================================

func (c *Context) JSON(status int, data interface{}) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	return json.NewEncoder(c.Writer).Encode(data)
}

func (c *Context) Bind(v interface{}) error {
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func (c *Context) Context() context.Context {
	return c.Request.Context()
}

// ================================
// Context Values
// ================================

func (c *Context) Set(key string, val interface{}) {
	c.values[key] = val
}

func (c *Context) Get(key string) interface{} {
	return c.values[key]
}

func (c *Context) GetString(key string) string {
	if v, ok := c.values[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ================================
// Response Helpers
// ================================

func Success(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

func Error(msg string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error":   msg,
	}
}
