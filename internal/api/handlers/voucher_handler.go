package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/quiqxiq/roskit/internal/services"
)

type VoucherHandler struct {
	svc         *services.VoucherService
	templateSvc *services.TemplateService
	hotspotSvc  *services.HotspotService
	settingsRepo repository.SettingsRepository
}

func NewVoucherHandler(
	svc *services.VoucherService,
	templateSvc *services.TemplateService,
	hotspotSvc *services.HotspotService,
	settingsRepo repository.SettingsRepository,
) *VoucherHandler {
	return &VoucherHandler{
		svc:          svc,
		templateSvc:  templateSvc,
		hotspotSvc:   hotspotSvc,
		settingsRepo: settingsRepo,
	}
}

func tenantLogoURL(logoPath string) string {
	if logoPath == "" {
		return ""
	}
	return "/api/v1/settings/logo"
}

func (h *VoucherHandler) Generate(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	var params services.VoucherGenerateParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	result, err := h.svc.GenerateVoucher(c.Request.Context(), routerID, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

type cacheVoucherRequest struct {
	Gencode string `json:"gencode" binding:"required,min=4,max=64"`
}

func (h *VoucherHandler) CacheVoucher(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	var req cacheVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "gencode is required"})
		return
	}

	result, err := h.svc.GetCachedVouchers(c.Request.Context(), routerID, req.Gencode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "voucher session not found or expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *VoucherHandler) RecordSale(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	var params services.RecordSaleParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	if err := h.svc.RecordSale(c.Request.Context(), routerID, params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "sale recorded"}, "error": nil})
}

func (h *VoucherHandler) ImportSales(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	result, err := h.svc.ImportSalesFromRouterOS(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *VoucherHandler) PrintData(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	gencode := c.Query("gencode")
	if gencode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "gencode query parameter is required"})
		return
	}

	result, err := h.svc.GetCachedVouchers(c.Request.Context(), routerID, gencode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "voucher session not found or expired"})
		return
	}

	settings, err := h.settingsRepo.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"vouchers": result.Vouchers,
			"router_info": gin.H{
				"hotspot_name": settings.HotspotName,
				"dns_name":     settings.DNSName,
				"currency":     settings.Currency,
				"phone":        settings.Phone,
				"email":        settings.Email,
				"info_lp":      settings.InfoLP,
			},
		},
		"error": nil,
	})
}

type printVouchersRequest struct {
	Usernames    []string `json:"usernames" binding:"omitempty,max=500,dive,min=1,max=64"`
	Comment      string   `json:"comment" binding:"omitempty,max=200"`
	TemplateType string   `json:"template_type" binding:"omitempty,oneof=default small thermal"`
}

func (h *VoucherHandler) PrintVouchers(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	var req printVouchersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}
	if len(req.Usernames) == 0 && req.Comment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "usernames or comment is required"})
		return
	}

	templateType := req.TemplateType
	if templateType == "" {
		templateType = "default"
	}

	ctx := c.Request.Context()

	if _, err := h.svc.GetRouterInfo(ctx, routerID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "router not found"})
		return
	}

	allUsers, err := h.hotspotSvc.ListUsers(ctx, routerID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to fetch hotspot users"})
		return
	}

	resolved, err := h.svc.ResolveVoucherPrintData(ctx, routerID, allUsers, req.Usernames, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	if len(resolved) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "no matching vouchers found"})
		return
	}

	settings, err := h.settingsRepo.Get(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get settings"})
		return
	}

	routerParams := services.RouterVoucherParams{
		HotspotName: settings.HotspotName,
		DNSName:     settings.DNSName,
		Logo:        tenantLogoURL(settings.LogoPath),
		Currency:    settings.Currency,
	}

	page, err := h.templateSvc.RenderFromUsers(ctx, templateType, resolved, routerParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}
