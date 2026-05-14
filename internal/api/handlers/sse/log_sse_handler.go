package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
)

type LogSSEHandler struct {
	bridge *service.Bridge
}

func NewLogSSEHandler(bridge *service.Bridge) *LogSSEHandler {
	return &LogSSEHandler{bridge: bridge}
}

func (h *LogSSEHandler) StreamAll(c *gin.Context)     { h.stream(c, "") }
func (h *LogSSEHandler) StreamHotspot(c *gin.Context) { h.stream(c, "hotspot") }
func (h *LogSSEHandler) StreamPPP(c *gin.Context)     { h.stream(c, "ppp") }

func (h *LogSSEHandler) stream(c *gin.Context, filter string) {
	routerIDStr := c.Param("routerId")
	if _, err := strconv.ParseUint(routerIDStr, 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid router id"})
		return
	}

	ctx := c.Request.Context()

	logCh, err := h.bridge.LogStream(ctx, routerIDStr, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open log stream"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	label := filter
	if label == "" {
		label = "all"
	}
	fmt.Fprintf(c.Writer, ": connected to %s log stream\n\n", label)
	c.Writer.Flush()

	keepAlive := time.NewTicker(sseKeepAliveInterval)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-keepAlive.C:
			if _, err := fmt.Fprintf(c.Writer, ": keep-alive\n\n"); err != nil {
				return
			}
			c.Writer.Flush()

		case entry, ok := <-logCh:
			if !ok {
				return
			}
			event := map[string]any{
				"router_id":   routerIDStr,
				"measurement": "log",
				"type":        "log",
				"fields": map[string]any{
					"message": entry["message"],
					"topics":  entry["topics"],
					"time":    entry["time"],
					"filter":  label,
				},
				"timestamp": time.Now().Format(time.RFC3339),
			}
			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}