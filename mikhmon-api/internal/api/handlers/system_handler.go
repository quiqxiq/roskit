package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/services"
)

type SystemHandler struct {
	svc *services.SystemService
}

func NewSystemHandler(svc *services.SystemService) *SystemHandler {
	return &SystemHandler{svc: svc}
}

func (h *SystemHandler) GetSystemResource(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.svc.GetSystemResource(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get system resource"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *SystemHandler) GetSystemLog(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	result, err := h.svc.GetSystemLog(c.Request.Context(), routerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get system log"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *SystemHandler) GetSystemClock(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.svc.GetSystemClock(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get system clock"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *SystemHandler) GetSystemIdentity(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.svc.GetSystemIdentity(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get system identity"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *SystemHandler) GetRouterboard(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.svc.GetRouterboard(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get routerboard info"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}


func (h *SystemHandler) GetDashboard(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.svc.GetDashboard(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get dashboard"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *SystemHandler) Reboot(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	if err := h.svc.Reboot(c.Request.Context(), routerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "reboot failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "reboot initiated"}, "error": nil})
}

func (h *SystemHandler) Shutdown(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	if err := h.svc.Shutdown(c.Request.Context(), routerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "shutdown failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "shutdown initiated"}, "error": nil})
}

func (h *SystemHandler) GetExpireMonitor(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.svc.GetExpireMonitorStatus(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get expire monitor status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

type deployMonitorRequest struct {
	Interval string `json:"interval" binding:"required"`
}

func (h *SystemHandler) DeployExpireMonitor(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	var req deployMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "interval is required"})
		return
	}
	if err := h.svc.DeployExpireMonitor(c.Request.Context(), routerID, req.Interval); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": fmt.Sprintf("deploy failed: %s", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "expire monitor deployed"}, "error": nil})
}

func (h *SystemHandler) RemoveExpireMonitor(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	if err := h.svc.RemoveExpireMonitor(c.Request.Context(), routerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": fmt.Sprintf("remove failed: %s", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "expire monitor removed"}, "error": nil})
}

func (h *SystemHandler) ListSchedulers(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.svc.ListSchedulers(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list schedulers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

// GetSystemResourceHistory returns historical CPU/memory metrics from InfluxDB.
// Query params:
//   - range: time window — 15m | 1h | 6h | 12h | 24h | 7d | 30d  (default: 1h)
//   - step:  bucket size — 30s | 1m | 5m | 15m | 30m | 1h | 6h | 12h | 1d (default: auto)
func (h *SystemHandler) GetSystemResourceHistory(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	rangeStr := c.DefaultQuery("range", "1h")
	step := c.Query("step") // empty = auto

	result, err := h.svc.GetSystemResourceHistory(c.Request.Context(), routerID, rangeStr, step)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": fmt.Sprintf("failed to query history: %s", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}
