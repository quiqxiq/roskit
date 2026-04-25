package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/services"
)

type VoucherHandler struct {
	svc *services.VoucherService
}

func NewVoucherHandler(svc *services.VoucherService) *VoucherHandler {
	return &VoucherHandler{svc: svc}
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
	Gencode string `json:"gencode" binding:"required"`
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

	router, err := h.svc.GetRouterInfo(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to get router info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"vouchers": result.Vouchers,
			"router_info": gin.H{
				"hotspot_name": router.HotspotName,
				"dns_name":     router.DNSName,
				"currency":     router.Currency,
				"phone":        router.Phone,
				"email":        router.Email,
				"info_lp":      router.InfoLP,
			},
		},
		"error": nil,
	})
}
