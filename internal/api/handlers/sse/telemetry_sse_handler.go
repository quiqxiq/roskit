package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
)

const sseKeepAliveInterval = 30 * time.Second

type TelemetrySSEHandler struct {
	sub    pubsub.Subscriber
	bridge *service.Bridge
}

func NewTelemetrySSEHandler(sub pubsub.Subscriber, bridge *service.Bridge) *TelemetrySSEHandler {
	return &TelemetrySSEHandler{sub: sub, bridge: bridge}
}

func (h *TelemetrySSEHandler) Stream(measurement string) gin.HandlerFunc {
	return func(c *gin.Context) {
		routerIDStr := c.Param("routerId")
		routerID, err := strconv.ParseUint(routerIDStr, 10, 64)
		if err != nil || routerID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
			return
		}

		iface := c.Param("iface")

		channel := pubsub.FormatPubSubChannel(fmt.Sprintf("%d", routerID))
		ctx := c.Request.Context()

		msgCh, err := h.sub.Subscribe(ctx, channel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to subscribe to telemetry stream"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Header("Access-Control-Allow-Origin", "*")

		fmt.Fprintf(c.Writer, ": connected to %s telemetry stream\n\n", measurement)
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

			case msg, ok := <-msgCh:
				if !ok {
					return
				}

			var event struct {
				Measurement string            `json:"measurement"`
				Fields      map[string]string `json:"fields"`
			}

			if err := json.Unmarshal(msg.Payload, &event); err != nil {
				continue
			}

			if event.Measurement != measurement {
				continue
			}

			if iface != "" && event.Fields["interface"] != iface && event.Fields["name"] != iface {
				continue
			}

				if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(msg.Payload)); err != nil {
					return
				}
				c.Writer.Flush()
			}
		}
	}
}

func (h *TelemetrySSEHandler) StreamInterface(c *gin.Context) {
	routerIDStr := c.Param("routerId")
	if _, err := strconv.ParseUint(routerIDStr, 10, 64); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	iface := c.Param("iface")
	if iface == "" {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "iface is required"})
		return
	}

	ctx := c.Request.Context()

	ch, err := h.bridge.InterfaceTrafficStream(ctx, routerIDStr, iface)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(c.Writer, ": connected to interface traffic stream for %s\n\n", iface)
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

		case fields, ok := <-ch:
			if !ok {
				return
			}

			event := map[string]any{
				"router_id":   routerIDStr,
				"measurement": "interface_traffic",
				"type":        "update",
				"fields":      fields,
				"timestamp":   time.Now().Format(time.RFC3339),
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
