package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roskit/internal/services"
)

type StatusHandler struct {
	svc *services.StatusService
}

func NewStatusHandler(svc *services.StatusService) *StatusHandler {
	return &StatusHandler{svc: svc}
}

func (h *StatusHandler) GetUserStatus(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	sessionName := c.Query("router")
	mac := c.Query("mac")
	if sessionName == "" || mac == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "router and mac parameters required"})
		return
	}
	status, err := h.svc.GetUserStatus(c.Request.Context(), tenantID, sessionName, mac)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status, "error": nil})
}
