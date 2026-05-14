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

type PingSSEHandler struct {
	bridge *service.Bridge
}

func NewPingSSEHandler(bridge *service.Bridge) *PingSSEHandler {
	return &PingSSEHandler{bridge: bridge}
}

// Stream handles GET /routers/:routerId/sse/ping?address=X.X.X.X[&count=N]
// Streams RouterOS ping results as SSE events. If count is omitted or 0, pings
// continuously until the client disconnects.
func (h *PingSSEHandler) Stream(c *gin.Context) {
	routerIDStr := c.Param("routerId")
	if _, err := strconv.ParseUint(routerIDStr, 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "address is required"})
		return
	}

	count := 0
	if countStr := c.Query("count"); countStr != "" {
		n, err := strconv.Atoi(countStr)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid count"})
			return
		}
		count = n
	}

	ctx := c.Request.Context()
	ch, err := h.bridge.PingStream(ctx, routerIDStr, address, count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(c.Writer, ": ping stream started for %s\n\n", address)
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

		case result, ok := <-ch:
			if !ok {
				fmt.Fprintf(c.Writer, "event: done\ndata: {}\n\n")
				c.Writer.Flush()
				return
			}

			event := map[string]any{
				"router_id":   routerIDStr,
				"measurement": "ping",
				"fields": map[string]any{
					"address":     address,
					"seq":         result.Seq,
					"host":        result.Host,
					"size":        result.Size,
					"time":        result.Time,
					"status":      result.Status,
					"sent":        result.Sent,
					"received":    result.Received,
					"packet_loss": result.PacketLoss,
				},
				"timestamp": result.At.Format(time.RFC3339),
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
