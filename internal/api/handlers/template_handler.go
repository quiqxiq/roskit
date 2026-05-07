package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/services"
)

type TemplateHandler struct {
	svc          *services.TemplateService
	voucherSvc   *services.VoucherService
	settingsRepo repository.TenantSettingsRepository
}

func NewTemplateHandler(
	svc *services.TemplateService,
	voucherSvc *services.VoucherService,
	settingsRepo repository.TenantSettingsRepository,
) *TemplateHandler {
	return &TemplateHandler{
		svc:          svc,
		voucherSvc:   voucherSvc,
		settingsRepo: settingsRepo,
	}
}

// scopeFromCtx returns either tenantID pointer (tenant scope) or nil (platform/global scope).
// Determined by role: superadmin without X-Tenant-Slug → global; otherwise tenant.
func (h *TemplateHandler) scopeFromCtx(c *gin.Context) (*uint, bool) {
	tenantID, ok := optionalTenantIDFromCtx(c)
	if !ok {
		// platform scope (superadmin operating on global defaults)
		return nil, true
	}
	tid := tenantID
	return &tid, true
}

func (h *TemplateHandler) Create(c *gin.Context) {
	scope, _ := h.scopeFromCtx(c)
	var req services.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	result, err := h.svc.Create(c.Request.Context(), scope, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

func (h *TemplateHandler) List(c *gin.Context) {
	tenantID, ok := optionalTenantIDFromCtx(c)
	if !ok {
		// platform scope: list global defaults
		results, err := h.svc.ListGlobal(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list templates"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": results, "error": nil})
		return
	}

	results, err := h.svc.List(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "error": nil})
}

func (h *TemplateHandler) Get(c *gin.Context) {
	scope, _ := h.scopeFromCtx(c)
	id, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid template id"})
		return
	}

	result, err := h.svc.GetByID(c.Request.Context(), scope, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *TemplateHandler) Update(c *gin.Context) {
	scope, _ := h.scopeFromCtx(c)
	id, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid template id"})
		return
	}

	var req services.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	result, err := h.svc.Update(c.Request.Context(), scope, uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *TemplateHandler) Delete(c *gin.Context) {
	scope, _ := h.scopeFromCtx(c)
	id, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid template id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), scope, uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "template deleted"}, "error": nil})
}

func (h *TemplateHandler) SeedDefaults(c *gin.Context) {
	var seedErr error
	if c.Query("force") == "true" {
		seedErr = h.svc.ForceSeedDefaults(c.Request.Context())
	} else {
		seedErr = h.svc.SeedDefaults(c.Request.Context())
	}
	if seedErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": seedErr.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"message": "default templates seeded"}, "error": nil})
}

type renderRequest struct {
	Gencode      string `json:"gencode" binding:"required,min=4,max=64"`
	TemplateType string `json:"template_type" binding:"required,oneof=default small thermal"`
	Profile      string `json:"profile" binding:"omitempty,max=64"`
	Validity     string `json:"validity" binding:"omitempty,max=64"`
	TimeLimit    string `json:"time_limit" binding:"omitempty,max=64"`
	DataLimit    string `json:"data_limit" binding:"omitempty,max=64"`
	Price        string `json:"price" binding:"omitempty,max=64"`
	Comment      string `json:"comment" binding:"omitempty,max=200"`
	UserMode     string `json:"user_mode" binding:"omitempty,max=32"`
	Logo         string `json:"logo" binding:"omitempty,max=255"`
	RouterID     uint   `json:"router_id" binding:"required,min=1"`
}

// Render renders a template type against cached vouchers identified by gencode.
// Tenant-scoped: settings come from tenant_settings, vouchers from per-router cache.
func (h *TemplateHandler) Render(c *gin.Context) {
	tenantID, ok := tenantIDFromCtx(c)
	if !ok {
		return
	}

	var req renderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	cached, err := h.voucherSvc.GetCachedVouchers(c.Request.Context(), req.RouterID, req.Gencode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "voucher session not found or expired"})
		return
	}

	if _, err := h.voucherSvc.GetRouterInfo(c.Request.Context(), tenantID, req.RouterID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get router info"})
		return
	}

	settings, err := h.settingsRepo.GetByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get tenant settings"})
		return
	}

	logo := req.Logo
	if logo == "" {
		logo = tenantLogoURL(tenantID, settings.LogoPath)
	}

	params := services.RenderParams{
		HotspotName: settings.HotspotName,
		DNSName:     settings.DNSName,
		Currency:    settings.Currency,
		Logo:        logo,
		UserMode:    req.UserMode,
		Profile:     req.Profile,
		Validity:    req.Validity,
		TimeLimit:   req.TimeLimit,
		DataLimit:   req.DataLimit,
		Price:       req.Price,
		Comment:     req.Comment,
	}

	vouchers := make([]roskitservice.GeneratedVoucher, len(cached.Vouchers))
	copy(vouchers, cached.Vouchers)

	page, err := h.svc.RenderPage(c.Request.Context(), tenantID, req.TemplateType, vouchers, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

var _ models.User // keep models import alive if not directly used
