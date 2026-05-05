package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/services"
)

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "record not found") || strings.Contains(err.Error(), "not found")
}

func parseRouterID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid router id")
	}
	return uint(id), nil
}

type RouterHandler struct {
	svc *services.RouterService
}

func NewRouterHandler(svc *services.RouterService) *RouterHandler {
	return &RouterHandler{svc: svc}
}

func (h *RouterHandler) Create(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}

	var req services.CreateRouterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	result, err := h.svc.CreateRouter(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

func (h *RouterHandler) Get(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	result, err := h.svc.GetRouter(c.Request.Context(), tenantID, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *RouterHandler) List(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	results, err := h.svc.ListRouters(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list routers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "error": nil})
}

func (h *RouterHandler) Update(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	var req services.UpdateRouterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	result, err := h.svc.UpdateRouter(c.Request.Context(), tenantID, uint(id), req)
	if err != nil {
		if isNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *RouterHandler) Delete(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	if err := h.svc.DeleteRouter(c.Request.Context(), tenantID, uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "router deleted"}, "error": nil})
}

func (h *RouterHandler) TestConnection(c *gin.Context) {
	var req struct {
		IPAddress   string `json:"ip_address"`
		APIPort     int    `json:"api_port"`
		APIUsername string `json:"api_username"`
		Password    string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	if req.IPAddress == "" || req.APIUsername == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "ip_address, api_username, and password are required"})
		return
	}

	port := req.APIPort
	if port == 0 {
		port = 8728
	}

	result, err := h.svc.TestConnection(c.Request.Context(), req.IPAddress, port, req.APIUsername, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "connection test failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

type migrateRequest struct {
	FilePath string `json:"file_path" binding:"required"`
}

func (h *RouterHandler) MigrateConfig(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	var req migrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "file_path is required"})
		return
	}

	result, err := h.svc.MigrateFromConfigPHP(c.Request.Context(), tenantID, req.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}
