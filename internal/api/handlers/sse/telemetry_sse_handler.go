package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
)

const sseKeepAliveInterval = 30 * time.Second

type TelemetrySSEHandler struct {
	sub pubsub.Subscriber
}

func NewTelemetrySSEHandler(sub pubsub.Subscriber) *TelemetrySSEHandler {
	return &TelemetrySSEHandler{sub: sub}
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

		channel := cache.FormatPubSubChannel(fmt.Sprintf("%d", routerID))
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
