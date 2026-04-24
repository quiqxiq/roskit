package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/services"
)

type HotspotHandler struct {
	svc *services.HotspotService
}

func NewHotspotHandler(svc *services.HotspotService) *HotspotHandler {
	return &HotspotHandler{svc: svc}
}

func parseRouterID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid router id")
	}
	return uint(id), nil
}

func (h *HotspotHandler) ListUsers(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	profile := c.Query("profile")
	result, err := h.svc.ListUsers(c.Request.Context(), routerID, profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list users"})
		return
	}

	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) GetUser(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	idOrName := c.Param("id")
	result, err := h.svc.GetUser(c.Request.Context(), routerID, idOrName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) GetUserCount(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	profile := c.Query("profile")
	count, err := h.svc.GetUserCount(c.Request.Context(), routerID, profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get user count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": map[string]int{"count": count}, "error": nil})
}

func (h *HotspotHandler) AddUser(c *gin.Context) {
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

	result, err := h.svc.AddUser(c.Request.Context(), routerID, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) UpdateUser(c *gin.Context) {
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

	result, err := h.svc.UpdateUser(c.Request.Context(), routerID, id, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) RemoveUser(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.RemoveUser(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "user removed"}, "error": nil})
}

func (h *HotspotHandler) ListProfiles(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	result, err := h.svc.ListProfiles(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list profiles"})
		return
	}

	if result == nil {
		result = []services.ProfileWithMeta{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) GetProfile(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	idOrName := c.Param("id")
	result, err := h.svc.GetProfile(c.Request.Context(), routerID, idOrName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) AddProfile(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	var params services.ProfileParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	result, err := h.svc.AddProfile(c.Request.Context(), routerID, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) UpdateProfile(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")

	var params services.ProfileParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	result, err := h.svc.UpdateProfile(c.Request.Context(), routerID, id, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) RemoveProfile(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.RemoveProfile(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "profile removed"}, "error": nil})
}

func (h *HotspotHandler) ListActive(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	server := c.Query("server")
	result, err := h.svc.ListActive(c.Request.Context(), routerID, server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list active sessions"})
		return
	}

	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) RemoveActive(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.RemoveActive(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "active session removed"}, "error": nil})
}

func (h *HotspotHandler) DisconnectUser(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.DisconnectUser(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "user disconnected"}, "error": nil})
}

func (h *HotspotHandler) ListHosts(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	result, err := h.svc.ListHosts(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list hosts"})
		return
	}

	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) RemoveHost(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.RemoveHost(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "host removed"}, "error": nil})
}

func (h *HotspotHandler) ListServers(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	result, err := h.svc.ListServers(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list servers"})
		return
	}

	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) ListCookies(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	result, err := h.svc.ListCookies(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list cookies"})
		return
	}

	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) RemoveCookie(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.RemoveCookie(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "cookie removed"}, "error": nil})
}

func (h *HotspotHandler) ListIPBindings(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	result, err := h.svc.ListIPBindings(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list IP bindings"})
		return
	}

	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) AddIPBinding(c *gin.Context) {
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

	result, err := h.svc.AddIPBinding(c.Request.Context(), routerID, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

func (h *HotspotHandler) UpdateIPBinding(c *gin.Context) {
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

	if err := h.svc.UpdateIPBinding(c.Request.Context(), routerID, id, params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "IP binding updated"}, "error": nil})
}

func (h *HotspotHandler) RemoveIPBinding(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.RemoveIPBinding(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "IP binding removed"}, "error": nil})
}

func (h *HotspotHandler) EnableIPBinding(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.EnableIPBinding(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "IP binding enabled"}, "error": nil})
}

func (h *HotspotHandler) DisableIPBinding(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.DisableIPBinding(c.Request.Context(), routerID, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "IP binding disabled"}, "error": nil})
}

func (h *HotspotHandler) ExportUsers(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	profile := c.Query("profile")
	format := c.DefaultQuery("format", "script")

	output, err := h.svc.ExportUsers(c.Request.Context(), routerID, profile, format)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=users_%d.csv", routerID))
		c.String(http.StatusOK, output)
	default:
		c.Header("Content-Type", "text/plain")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=users_%d.rsc", routerID))
		c.String(http.StatusOK, output)
	}
}
