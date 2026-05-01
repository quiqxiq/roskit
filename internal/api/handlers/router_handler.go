package handlers

import (
	"fmt"
	"io"
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

// routerLogoURL converts a stored LogoPath to a web-accessible URL.
// Returns empty string if no logo is configured.
func routerLogoURL(routerID uint, logoPath string) string {
	if logoPath == "" {
		return ""
	}
	return fmt.Sprintf("/api/v1/routers/%d/logo", routerID)
}

type RouterHandler struct {
	svc *services.RouterService
}

func NewRouterHandler(svc *services.RouterService) *RouterHandler {
	return &RouterHandler{svc: svc}
}

func (h *RouterHandler) Create(c *gin.Context) {
	var req services.CreateRouterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	result, err := h.svc.CreateRouter(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

func (h *RouterHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	result, err := h.svc.GetRouter(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *RouterHandler) List(c *gin.Context) {
	results, err := h.svc.ListRouters(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list routers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "error": nil})
}

func (h *RouterHandler) Update(c *gin.Context) {
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

	result, err := h.svc.UpdateRouter(c.Request.Context(), uint(id), req)
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
	id, err := strconv.ParseUint(c.Param("routerId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	if err := h.svc.DeleteRouter(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "router deleted"}, "error": nil})
}

func (h *RouterHandler) TestConnection(c *gin.Context) {
	var req struct {
		IP       string `json:"ip"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	if req.IP == "" || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "ip, username, and password are required"})
		return
	}

	result, err := h.svc.TestConnection(c.Request.Context(), req.IP, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "connection test failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *RouterHandler) UploadLogo(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	file, err := c.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "logo file required"})
		return
	}
	if file.Size > 1<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "file too large, max 1MB"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".png") {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "only PNG files allowed"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to read file"})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to read file"})
		return
	}
	if err := h.svc.UploadLogo(c.Request.Context(), routerID, data, file.Filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to upload logo"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "logo uploaded"}, "error": nil})
}

func (h *RouterHandler) GetLogo(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	path, err := h.svc.GetLogoPath(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "logo not found"})
		return
	}
	c.File(path)
}

type migrateRequest struct {
	FilePath string `json:"file_path" binding:"required"`
}

func (h *RouterHandler) MigrateConfig(c *gin.Context) {
	var req migrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "file_path is required"})
		return
	}

	result, err := h.svc.MigrateFromConfigPHP(c.Request.Context(), req.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}
