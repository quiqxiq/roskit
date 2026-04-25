package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	roskitservice "github.com/quiqxiq/roskit/internal/roskit/adapter/service"
)

type NetworkHandler struct {
	bridge *roskitservice.Bridge
}

func NewNetworkHandler(bridge *roskitservice.Bridge) *NetworkHandler {
	return &NetworkHandler{bridge: bridge}
}

func (h *NetworkHandler) ListInterfaces(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.bridge.ListInterfaces(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list interfaces"})
		return
	}
	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *NetworkHandler) GetInterfaceTraffic(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	iface := c.Param("iface")
	data, err := h.bridge.GetCachedSnapshot(c.Request.Context(), fmt.Sprintf("roskit:%d:interface_traffic:%s", routerID, iface))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"data": nil, "error": "traffic data not found for interface"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "error": nil})
}

func (h *NetworkHandler) ListPools(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.bridge.ListIPPools(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list IP pools"})
		return
	}
	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *NetworkHandler) ListQueues(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.bridge.ListParentQueues(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list queues"})
		return
	}
	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *NetworkHandler) ListNATRules(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	result, err := h.bridge.ListNATRules(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list NAT rules"})
		return
	}
	if result == nil {
		result = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "error": nil})
}

func (h *NetworkHandler) ListDHCPLeases(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	items, err := h.bridge.ListDHCPLeases(c.Request.Context(), fmt.Sprintf("%d", routerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to list DHCP leases"})
		return
	}
	if items == nil {
		items = []map[string]string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "error": nil})
}

func (h *NetworkHandler) ReleaseDHCPLease(c *gin.Context) {
	routerID, err := parseRouterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	id := c.Param("id")
	_, err = h.bridge.Run(c.Request.Context(), fmt.Sprintf("%d", routerID), "/ip/dhcp-server/lease/release", "=.id="+id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "DHCP lease released"}, "error": nil})
}
