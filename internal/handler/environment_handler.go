package handler

import (
	"net/http"
	"strconv"

	"test-management-service/internal/service"

	"github.com/gin-gonic/gin"
)

// EnvironmentHandler handles HTTP requests for environment management
type EnvironmentHandler struct {
	envService service.EnvironmentService
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(envService service.EnvironmentService) *EnvironmentHandler {
	return &EnvironmentHandler{
		envService: envService,
	}
}

// RegisterRoutes registers all environment-related routes
func (h *EnvironmentHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v2/environments")
	{
		// Environment CRUD operations
		api.POST("", h.CreateEnvironment)
		api.GET("", h.ListEnvironments)
		api.GET("/active", h.GetActiveEnvironment)
		api.GET("/:id", h.GetEnvironment)
		api.PUT("/:id", h.UpdateEnvironment)
		api.DELETE("/:id", h.DeleteEnvironment)
		api.POST("/:id/activate", h.ActivateEnvironment)

		// Variable operations
		api.GET("/:id/variables", h.GetVariables)
		api.GET("/:id/variables/:key", h.GetVariable)
		api.PUT("/:id/variables/:key", h.SetVariable)
		api.DELETE("/:id/variables/:key", h.DeleteVariable)
	}
}

// ===== Environment Handlers =====

// CreateEnvironment creates a new environment
// POST /api/v2/environments
func (h *EnvironmentHandler) CreateEnvironment(c *gin.Context) {
	var req service.CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := h.envService.CreateEnvironment(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, env)
}

// ListEnvironments lists all environments with pagination
// GET /api/v2/environments?limit=20&offset=0
func (h *EnvironmentHandler) ListEnvironments(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	environments, total, err := h.envService.ListEnvironments(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   environments,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetEnvironment retrieves a specific environment by ID
// GET /api/v2/environments/:id
func (h *EnvironmentHandler) GetEnvironment(c *gin.Context) {
	envID := c.Param("id")

	env, err := h.envService.GetEnvironment(envID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if env == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "environment not found"})
		return
	}

	c.JSON(http.StatusOK, env)
}

// UpdateEnvironment updates an existing environment
// PUT /api/v2/environments/:id
func (h *EnvironmentHandler) UpdateEnvironment(c *gin.Context) {
	envID := c.Param("id")

	var req service.UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := h.envService.UpdateEnvironment(envID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, env)
}

// DeleteEnvironment deletes an environment
// DELETE /api/v2/environments/:id
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
	envID := c.Param("id")

	if err := h.envService.DeleteEnvironment(envID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "environment deleted"})
}

// GetActiveEnvironment retrieves the currently active environment
// GET /api/v2/environments/active
func (h *EnvironmentHandler) GetActiveEnvironment(c *gin.Context) {
	env, err := h.envService.GetActiveEnvironment()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if env == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active environment found"})
		return
	}

	c.JSON(http.StatusOK, env)
}

// ActivateEnvironment activates a specific environment
// POST /api/v2/environments/:id/activate
func (h *EnvironmentHandler) ActivateEnvironment(c *gin.Context) {
	envID := c.Param("id")

	err := h.envService.ActivateEnvironment(envID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "environment activated",
		"envId":   envID,
	})
}

// ===== Variable Handlers =====

// GetVariables retrieves all variables for an environment
// GET /api/v2/environments/:id/variables
func (h *EnvironmentHandler) GetVariables(c *gin.Context) {
	envID := c.Param("id")

	variables, err := h.envService.GetVariables(envID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, variables)
}

// GetVariable retrieves a specific variable by key
// GET /api/v2/environments/:id/variables/:key
func (h *EnvironmentHandler) GetVariable(c *gin.Context) {
	envID := c.Param("id")
	key := c.Param("key")

	value, err := h.envService.GetVariable(envID, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": value,
	})
}

// SetVariable creates or updates a variable
// PUT /api/v2/environments/:id/variables/:key
func (h *EnvironmentHandler) SetVariable(c *gin.Context) {
	envID := c.Param("id")
	key := c.Param("key")

	var req service.SetVariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.envService.SetVariable(envID, key, req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "variable updated",
		"key":     key,
		"value":   req.Value,
	})
}

// DeleteVariable deletes a variable
// DELETE /api/v2/environments/:id/variables/:key
func (h *EnvironmentHandler) DeleteVariable(c *gin.Context) {
	envID := c.Param("id")
	key := c.Param("key")

	if err := h.envService.DeleteVariable(envID, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "variable deleted"})
}
