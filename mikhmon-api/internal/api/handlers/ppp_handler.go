package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
)

type PPPHandler struct {
	bridge *roskitservice.Bridge
}

func NewPPPHandler(bridge *roskitservice.Bridge) *PPPHandler {
	return &PPPHandler{bridge: bridge}
}

func (h *PPPHandler) ListSecrets(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	items, err := h.bridge.ListPPPSecrets(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list PPP secrets"})
		return
	}
	if items == nil {
		items = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "error": nil})
}

func (h *PPPHandler) AddSecret(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	var params map[string]string
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}
	_, err = h.bridge.AddPPPSecret(c.Request.Context(), fmt.Sprintf("%d", routerID), params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"message": "PPP secret added"}, "error": nil})
}

func (h *PPPHandler) UpdateSecret(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	id := c.Param("id")
	var params map[string]string
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}
	if err := h.bridge.SetPPPSecret(c.Request.Context(), fmt.Sprintf("%d", routerID), id, params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "PPP secret updated"}, "error": nil})
}

func (h *PPPHandler) RemoveSecret(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	id := c.Param("id")
	if err := h.bridge.RemovePPPSecret(c.Request.Context(), fmt.Sprintf("%d", routerID), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "PPP secret removed"}, "error": nil})
}

func (h *PPPHandler) ListActive(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	items, err := h.bridge.ListPPPActive(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list PPP active sessions"})
		return
	}
	if items == nil {
		items = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "error": nil})
}

func (h *PPPHandler) DisconnectActive(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	id := c.Param("id")
	if err := h.bridge.RemovePPPActive(c.Request.Context(), fmt.Sprintf("%d", routerID), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "PPP active session disconnected"}, "error": nil})
}

func (h *PPPHandler) ListProfiles(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.bridge.ListPPPProfiles(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list PPP profiles"})
		return
	}
	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}
