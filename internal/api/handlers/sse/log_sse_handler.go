package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/quiqxiq/roskit/internal/roskit/adapter/service"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
)

type LogSSEHandler struct {
	sub   pubsub.Subscriber
	bridge *service.Bridge
}

func NewLogSSEHandler(sub pubsub.Subscriber, bridge *service.Bridge) *LogSSEHandler {
	return &LogSSEHandler{sub: sub, bridge: bridge}
}

func (h *LogSSEHandler) StreamAll(c *gin.Context) {
	h.stream(c, "all")
}

func (h *LogSSEHandler) StreamHotspot(c *gin.Context) {
	h.stream(c, "hotspot")
}

func (h *LogSSEHandler) StreamPPP(c *gin.Context) {
	h.stream(c, "ppp")
}

func (h *LogSSEHandler) stream(c *gin.Context, filter string) {
	routerIDStr := c.Param("routerId")
	routerID, err := strconv.ParseUint(routerIDStr, 10, 64)
	if err != nil || routerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"data": nil, "error": "invalid router id"})
		return
	}

	channel := pubsub.FormatLogChannel(fmt.Sprintf("%d", routerID), filter)

	ctx := c.Request.Context()

	msgCh, err := h.sub.Subscribe(ctx, channel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"data": nil, "error": "failed to subscribe to log stream"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(c.Writer, ": connected to %s log stream\n\n", filter)
	c.Writer.Flush()

	h.sendInitialDump(c, fmt.Sprintf("%d", routerID), filter)

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
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(msg.Payload)); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

func (h *LogSSEHandler) sendInitialDump(c *gin.Context, routerID, filter string) {
	if h.bridge == nil {
		return
	}

	cmd := []string{"/log/print"}
	switch filter {
	case "hotspot":
		cmd = append(cmd, "?topics=hotspot,info")
	case "ppp":
		cmd = append(cmd, "?topics=pppoe,info")
	}

	ctx := c.Request.Context()
	reply, err := h.bridge.Run(ctx, routerID, cmd...)
	if err != nil {
		fmt.Fprintf(c.Writer, ": initial dump unavailable: %s\n\n", err.Error())
		c.Writer.Flush()
		return
	}

	now := time.Now().Format("2006-01-02T15:04:05Z07:00")
	count := 0
	for _, re := range reply.Re {
		msg := re.Map["message"]
		topics := re.Map["topics"]
		t := re.Map["time"]

		if filter == "hotspot" && !strings.Contains(topics, "hotspot") {
			continue
		}
		if filter == "ppp" && !strings.Contains(topics, "pppoe") {
			continue
		}

		event := map[string]interface{}{
			"router_id":   routerID,
			"measurement": "log",
			"type":        "log",
			"fields": map[string]interface{}{
				"message": msg,
				"topics":  topics,
				"time":    t,
				"filter":  filter,
			},
			"timestamp": now,
		}

		payload, err := json.Marshal(event)
		if err != nil {
			continue
		}

		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		count++
	}

	if count > 0 {
		c.Writer.Flush()
	}

	fmt.Fprintf(c.Writer, ": initial dump complete (%d entries)\n\n", count)
	c.Writer.Flush()
}
