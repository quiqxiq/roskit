package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
)

type QuickPrintHandler struct {
	bridge *roskitservice.Bridge
}

func NewQuickPrintHandler(bridge *roskitservice.Bridge) *QuickPrintHandler {
	return &QuickPrintHandler{bridge: bridge}
}

func (h *QuickPrintHandler) ListPackages(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	pkgs, err := h.bridge.ListQuickPrintPackages(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list quick print packages"})
		return
	}
	if pkgs == nil {
		pkgs = []*roskitservice.QuickPrintPackage{}
	}
	c.JSON(http.StatusOK, gin.H{"data": pkgs, "error": nil})
}

func (h *QuickPrintHandler) CreatePackage(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	var body struct {
		Name         string `json:"name" binding:"required"`
		Server       string `json:"server"`
		UserMode     string `json:"user_mode"`
		UserLength   string `json:"user_length"`
		Prefix       string `json:"prefix"`
		CharMode     string `json:"char_mode"`
		Profile      string `json:"profile"`
		TimeLimit    string `json:"time_limit"`
		DataLimit    string `json:"data_limit"`
		Comment      string `json:"comment"`
		Validity     string `json:"validity"`
		Price        string `json:"price"`
		SellingPrice string `json:"selling_price"`
		LockUser     string `json:"lock_user"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "name is required"})
		return
	}
	pkg := &roskitservice.QuickPrintPackage{
		Name:         body.Name,
		Server:       body.Server,
		UserMode:     body.UserMode,
		UserLength:   body.UserLength,
		Prefix:       body.Prefix,
		CharMode:     body.CharMode,
		Profile:      body.Profile,
		TimeLimit:    body.TimeLimit,
		DataLimit:    body.DataLimit,
		Comment:      body.Comment,
		Validity:     body.Validity,
		Price:        body.Price,
		SellingPrice: body.SellingPrice,
		LockUser:     body.LockUser,
	}
	if err := h.bridge.SaveQuickPrintPackage(c.Request.Context(), fmt.Sprintf("%d", routerID), pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"message": "quick print package created"}, "error": nil})
}

func (h *QuickPrintHandler) GetPackage(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	name := c.Param("name")
	pkg, err := h.bridge.GetQuickPrintPackage(c.Request.Context(), fmt.Sprintf("%d", routerID), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "package not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pkg, "error": nil})
}

func (h *QuickPrintHandler) UpdatePackage(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	id := c.Param("name")
	var body struct {
		Name         string `json:"name"`
		Server       string `json:"server"`
		UserMode     string `json:"user_mode"`
		UserLength   string `json:"user_length"`
		Prefix       string `json:"prefix"`
		CharMode     string `json:"char_mode"`
		Profile      string `json:"profile"`
		TimeLimit    string `json:"time_limit"`
		DataLimit    string `json:"data_limit"`
		Comment      string `json:"comment"`
		Validity     string `json:"validity"`
		Price        string `json:"price"`
		SellingPrice string `json:"selling_price"`
		LockUser     string `json:"lock_user"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}
	pkg := &roskitservice.QuickPrintPackage{
		Name:         body.Name,
		Server:       body.Server,
		UserMode:     body.UserMode,
		UserLength:   body.UserLength,
		Prefix:       body.Prefix,
		CharMode:     body.CharMode,
		Profile:      body.Profile,
		TimeLimit:    body.TimeLimit,
		DataLimit:    body.DataLimit,
		Comment:      body.Comment,
		Validity:     body.Validity,
		Price:        body.Price,
		SellingPrice: body.SellingPrice,
		LockUser:     body.LockUser,
	}
	if err := h.bridge.UpdateQuickPrintPackage(c.Request.Context(), fmt.Sprintf("%d", routerID), id, pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "quick print package updated"}, "error": nil})
}

func (h *QuickPrintHandler) RemovePackage(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	id := c.Param("name")
	if err := h.bridge.RemoveQuickPrintPackage(c.Request.Context(), fmt.Sprintf("%d", routerID), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "quick print package removed"}, "error": nil})
}
