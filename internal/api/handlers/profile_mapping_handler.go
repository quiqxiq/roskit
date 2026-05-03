package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roskit/internal/models"
	"github.com/quiqxiq/roskit/internal/repository"
)

type ProfileMappingHandler struct {
	repo repository.ProfilePriceMappingRepository
}

func NewProfileMappingHandler(repo repository.ProfilePriceMappingRepository) *ProfileMappingHandler {
	return &ProfileMappingHandler{repo: repo}
}

func (h *ProfileMappingHandler) List(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}

	mappings, err := h.repo.ListByRouter(c.Request.Context(), routerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list profile mappings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mappings, "error": nil})
}

type updateMappingRequest struct {
	Price        int64  `json:"price"`
	SellingPrice int64  `json:"selling_price"`
	Validity     string `json:"validity"`
	ExpMode      string `json:"exp_mode"`
	LockUser     *bool  `json:"lock_user"`
	LockServer   *bool  `json:"lock_server"`
}

func (h *ProfileMappingHandler) Update(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	profileName := c.Param("profileName")
	if profileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "profile name required"})
		return
	}

	var req updateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid request body"})
		return
	}

	mapping := &models.ProfilePriceMapping{
		RouterID:    routerID,
		ProfileName: profileName,
		Price:       req.Price,
		SellingPrice: req.SellingPrice,
		Validity:    req.Validity,
		ExpMode:     req.ExpMode,
	}
	if req.LockUser != nil {
		mapping.LockUser = *req.LockUser
	}
	if req.LockServer != nil {
		mapping.LockServer = *req.LockServer
	}

	if err := h.repo.Upsert(c.Request.Context(), mapping); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to update profile mapping"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mapping, "error": nil})
}

func (h *ProfileMappingHandler) Delete(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	profileName := c.Param("profileName")
	if profileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "profile name required"})
		return
	}

	if err := h.repo.DeleteByRouterAndProfile(c.Request.Context(), routerID, profileName); err != nil {
		if isNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "mapping not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to delete profile mapping"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "mapping deleted"}, "error": nil})
}

func parseUintParam(c *gin.Context, param string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
