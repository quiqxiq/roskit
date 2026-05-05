package handlers

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/services"
)

type TenantHandler struct {
	svc *services.TenantService
}

func NewTenantHandler(svc *services.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

// ====== Tenant-self endpoints (owner only) ======

func (h *TenantHandler) GetSelf(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	tenant, err := h.svc.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tenant, "error": nil})
}

type updateTenantNameRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *TenantHandler) UpdateSelfName(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	var req updateTenantNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "name is required"})
		return
	}
	if err := h.svc.UpdateName(c.Request.Context(), tenantID, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "tenant name updated"}, "error": nil})
}

func (h *TenantHandler) GetSelfSettings(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	settings, err := h.svc.GetSettings(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "settings not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings, "error": nil})
}

func (h *TenantHandler) UpdateSelfSettings(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	var req services.UpdateTenantSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}
	settings, err := h.svc.UpdateSettings(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings, "error": nil})
}

func (h *TenantHandler) UploadLogo(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
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
	if !isAllowedImageMIME(data) {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "only PNG, JPEG, or WebP images are allowed"})
		return
	}
	if _, err := h.svc.UploadLogo(c.Request.Context(), tenantID, data, file.Filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to upload logo"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "logo uploaded"}, "error": nil})
}

func (h *TenantHandler) GetLogo(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}
	path, err := h.svc.GetLogoPath(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "logo not found"})
		return
	}
	c.File(path)
}

func isAllowedImageMIME(data []byte) bool {
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

// ====== Platform admin endpoints (superadmin only) ======

func (h *TenantHandler) AdminList(c *gin.Context) {
	tenants, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list tenants"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tenants, "error": nil})
}

func (h *TenantHandler) AdminCreate(c *gin.Context) {
	var req services.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}
	tenant, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tenant, "error": nil})
}

func (h *TenantHandler) AdminGet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid tenant id"})
		return
	}
	tenant, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tenant, "error": nil})
}

func (h *TenantHandler) AdminUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid tenant id"})
		return
	}
	var req updateTenantNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "name is required"})
		return
	}
	if err := h.svc.UpdateName(c.Request.Context(), uint(id), req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "tenant updated"}, "error": nil})
}

func (h *TenantHandler) AdminHardDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid tenant id"})
		return
	}
	if err := h.svc.HardDelete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "tenant deleted"}, "error": nil})
}

func (h *TenantHandler) AdminSuspend(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid tenant id"})
		return
	}
	if err := h.svc.Suspend(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "tenant suspended"}, "error": nil})
}

func (h *TenantHandler) AdminActivate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid tenant id"})
		return
	}
	if err := h.svc.Activate(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "tenant activated"}, "error": nil})
}
