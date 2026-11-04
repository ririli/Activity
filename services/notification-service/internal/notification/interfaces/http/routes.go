package http

import (
	"activity/pkg/platform"
	"activity/services/notification-service/internal/notification/application"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func Routes(g *gin.Engine, h *application.Hub) {
	g.GET("/health/subscription", func(c *gin.Context) {
		if !h.Ready() {
			c.Status(503)
		} else {
			c.Status(200)
		}
	})
	g.GET("/api/notification/v1/barrage/stream", platform.Auth, func(c *gin.Context) {
		p := c.MustGet("principal").(platform.Principal)
		conn, e := h.Subscribe(p.SessionID)
		if e != nil {
			c.Header("Retry-After", "5")
			platform.Respond(c, nil, e)
			return
		}
		defer h.Unsubscribe(conn)
		c.Header("Content-Type", "text/event-stream; charset=utf-8")
		c.Header("Cache-Control", "no-cache, no-store")
		c.Header("X-Accel-Buffering", "no")
		rc := http.NewResponseController(c.Writer)
		write := func(s string) bool {
			_ = rc.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, e := fmt.Fprint(c.Writer, s); e != nil {
				return false
			}
			return rc.Flush() == nil
		}
		if !write("event: ready\ndata: {\"status\":\"connected\"}\n\n") {
			return
		}
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		verify := time.NewTicker(27*time.Second + time.Duration(time.Now().UnixNano()%3000000000))
		defer verify.Stop()
		availability := time.NewTicker(time.Second)
		defer availability.Stop()
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-availability.C:
				if !h.Ready() {
					write("event: unavailable\ndata: {}\n\n")
					return
				}
			case <-verify.C:
				if _, e := platform.Verify(c.Request.Context(), token); e != nil {
					write("event: auth_required\ndata: {}\n\n")
					return
				}
			case <-heartbeat.C:
				if !write(": heartbeat\n\n") {
					return
				}
			case m := <-conn.Messages:
				if !m.Expires.After(time.Now()) {
					continue
				}
				b, _ := json.Marshal(map[string]any{"message_id": m.ID, "display_name": m.Name, "item_name": m.ItemName, "quantity": m.Quantity, "expires_at": m.Expires})
				if !write("event: barrage\nid: " + m.ID + "\ndata: " + string(b) + "\n\n") {
					return
				}
			}
		}
	})
}
