package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/services"
)

type TemplateHandler struct {
	svc        *services.TemplateService
	voucherSvc *services.VoucherService
}

func NewTemplateHandler(svc *services.TemplateService, voucherSvc *services.VoucherService) *TemplateHandler {
	return &TemplateHandler{svc: svc, voucherSvc: voucherSvc}
}

func (h *TemplateHandler) Create(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	var req services.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	result, err := h.svc.Create(c.Request.Context(), routerID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result, "error": nil})
}

func (h *TemplateHandler) List(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	results, err := h.svc.List(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "error": nil})
}

func (h *TemplateHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid template id"})
		return
	}

	result, err := h.svc.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *TemplateHandler) Update(c *gin.Context) {
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

	result, err := h.svc.Update(c.Request.Context(), uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *TemplateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("templateId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid template id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "template deleted"}, "error": nil})
}

type renderRequest struct {
	Gencode      string `json:"gencode" binding:"required"`
	TemplateType string `json:"template_type" binding:"required"`
	Profile      string `json:"profile"`
	Validity     string `json:"validity"`
	TimeLimit    string `json:"time_limit"`
	DataLimit    string `json:"data_limit"`
	Price        string `json:"price"`
	Comment      string `json:"comment"`
	UserMode     string `json:"user_mode"` // "vc" or "up"
	Logo         string `json:"logo"`
}

// Render renders a template type against cached vouchers identified by gencode.
func (h *TemplateHandler) Render(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	var req renderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body: " + err.Error()})
		return
	}

	cached, err := h.voucherSvc.GetCachedVouchers(c.Request.Context(), routerID, req.Gencode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "voucher session not found or expired"})
		return
	}

	router, err := h.voucherSvc.GetRouterInfo(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get router info"})
		return
	}

	params := services.RenderParams{
		HotspotName: router.HotspotName,
		DNSName:     router.DNSName,
		Currency:    router.Currency,
		Logo:        req.Logo,
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

	rendered, err := h.svc.Render(c.Request.Context(), routerID, req.TemplateType, vouchers, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rendered, "error": nil})
}
