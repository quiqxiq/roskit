package handlers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/services"
)

type SettingsHandler struct {
	svc *services.SettingsService
}

func NewSettingsHandler(svc *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.svc.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to load settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings, "error": nil})
}

func (h *SettingsHandler) Update(c *gin.Context) {
	var req services.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}
	settings, err := h.svc.Update(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings, "error": nil})
}

func (h *SettingsHandler) UploadLogo(c *gin.Context) {
	file, err := c.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "logo file required"})
		return
	}
	if file.Size > 1<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "file too large, max 1MB"})
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
	if !isAllowedImageData(data) {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "only PNG, JPEG, or WebP images are allowed"})
		return
	}
	if _, err := h.svc.UploadLogo(c.Request.Context(), data, file.Filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to upload logo"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "logo uploaded"}, "error": nil})
}

func (h *SettingsHandler) GetLogo(c *gin.Context) {
	path, err := h.svc.GetLogoPath(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "logo not found"})
		return
	}
	c.File(path)
}

func isAllowedImageData(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return true
	}
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return true
	}
	if len(data) >= 12 && bytes.Equal(data[8:12], []byte("WEBP")) {
		return true
	}
	return false
}
