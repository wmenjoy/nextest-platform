package handler

import (
	"net/http"
	"strconv"

	"test-management-service/internal/service"

	"github.com/gin-gonic/gin"
)

// TestHandler HTTP处理器
type TestHandler struct {
	service service.TestService
}

// NewTestHandler 创建处理器
func NewTestHandler(service service.TestService) *TestHandler {
	return &TestHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *TestHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v2")
	{
		// Test cases
		api.POST("/tests", h.CreateTestCase)
		api.PUT("/tests/:id", h.UpdateTestCase)
		api.DELETE("/tests/:id", h.DeleteTestCase)
		api.GET("/tests/:id", h.GetTestCase)
		api.GET("/tests", h.ListTestCases)
		api.GET("/tests/search", h.SearchTestCases)
		api.GET("/tests/stats", h.GetTestStats)

		// Test tree (for Web UI)
		api.GET("/test-tree", h.GetTestTree)

		// Test groups
		api.POST("/groups", h.CreateTestGroup)
		api.PUT("/groups/:id", h.UpdateTestGroup)
		api.DELETE("/groups/:id", h.DeleteTestGroup)
		api.GET("/groups/:id", h.GetTestGroup)
		api.GET("/groups/tree", h.GetTestGroupTree)

		// Test execution
		api.POST("/tests/:id/execute", h.ExecuteTest)
		api.POST("/groups/:id/execute", h.ExecuteTestGroup)

		// Test results
		api.GET("/results/:id", h.GetTestResult)
		api.GET("/tests/:id/history", h.GetTestHistory)

		// Test runs
		api.GET("/runs/:id", h.GetTestRun)
		api.GET("/runs", h.ListTestRuns)
	}
}

// ===== Test Case Handlers =====

func (h *TestHandler) CreateTestCase(c *gin.Context) {
	var req service.CreateTestCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	testCase, err := h.service.CreateTestCase(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, testCase)
}

func (h *TestHandler) UpdateTestCase(c *gin.Context) {
	testID := c.Param("id")
	var req service.UpdateTestCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	testCase, err := h.service.UpdateTestCase(testID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, testCase)
}

func (h *TestHandler) DeleteTestCase(c *gin.Context) {
	testID := c.Param("id")
	if err := h.service.DeleteTestCase(testID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test case deleted"})
}

func (h *TestHandler) GetTestCase(c *gin.Context) {
	testID := c.Param("id")
	testCase, err := h.service.GetTestCase(testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if testCase == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test case not found"})
		return
	}

	c.JSON(http.StatusOK, testCase)
}

func (h *TestHandler) ListTestCases(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	testCases, total, err := h.service.ListTestCases(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  testCases,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

func (h *TestHandler) SearchTestCases(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	testCases, err := h.service.SearchTestCases(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, testCases)
}

// ===== Test Group Handlers =====

func (h *TestHandler) CreateTestGroup(c *gin.Context) {
	var req service.CreateTestGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.service.CreateTestGroup(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *TestHandler) UpdateTestGroup(c *gin.Context) {
	groupID := c.Param("id")
	var req service.UpdateTestGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.service.UpdateTestGroup(groupID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *TestHandler) DeleteTestGroup(c *gin.Context) {
	groupID := c.Param("id")
	if err := h.service.DeleteTestGroup(groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test group deleted"})
}

func (h *TestHandler) GetTestGroup(c *gin.Context) {
	groupID := c.Param("id")
	group, err := h.service.GetTestGroup(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if group == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *TestHandler) GetTestGroupTree(c *gin.Context) {
	tree, err := h.service.GetTestGroupTree()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tree)
}

// ===== Test Execution Handlers =====

func (h *TestHandler) ExecuteTest(c *gin.Context) {
	testID := c.Param("id")
	result, err := h.service.ExecuteTest(testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *TestHandler) ExecuteTestGroup(c *gin.Context) {
	groupID := c.Param("id")
	run, err := h.service.ExecuteTestGroup(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, run)
}

// ===== Test Result Handlers =====

func (h *TestHandler) GetTestResult(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid result id"})
		return
	}

	result, err := h.service.GetTestResult(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test result not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *TestHandler) GetTestHistory(c *gin.Context) {
	testID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	history, err := h.service.GetTestHistory(testID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

// ===== Test Run Handlers =====

func (h *TestHandler) GetTestRun(c *gin.Context) {
	runID := c.Param("id")
	run, err := h.service.GetTestRun(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if run == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test run not found"})
		return
	}

	c.JSON(http.StatusOK, run)
}

func (h *TestHandler) ListTestRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	runs, total, err := h.service.ListTestRuns(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   runs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ===== Web UI Specific Handlers =====

// GetTestTree returns the complete test tree with groups and tests
func (h *TestHandler) GetTestTree(c *gin.Context) {
	tree, err := h.service.GetTestGroupTree()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Enrich tree with test cases for each group
	allTests, _, err := h.service.ListTestCases(10000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Organize tests by group
	for i := range tree {
		for _, test := range allTests {
			if test.GroupID == tree[i].GroupID {
				tree[i].TestCases = append(tree[i].TestCases, test)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tree": tree,
		},
	})
}

// GetTestStats returns test statistics
func (h *TestHandler) GetTestStats(c *gin.Context) {
	// Get all test cases
	tests, total, err := h.service.ListTestCases(10000, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Calculate statistics
	stats := gin.H{
		"total":  total,
		"active": 0,
		"p0":     0,
		"p1":     0,
		"p2":     0,
	}

	for _, test := range tests {
		if test.Status == "active" {
			stats["active"] = stats["active"].(int) + 1
		}
		switch test.Priority {
		case "P0":
			stats["p0"] = stats["p0"].(int) + 1
		case "P1":
			stats["p1"] = stats["p1"].(int) + 1
		case "P2":
			stats["p2"] = stats["p2"].(int) + 1
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

