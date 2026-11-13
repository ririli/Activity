package http

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	app "activity/services/activity-service/internal/application"
	"github.com/gin-gonic/gin"
)

func HourlyRoutes(g *gin.Engine, s app.HourlyService) {
	a := g.Group("/internal/activity/v1", platform.Internal)
	a.POST("/hourly-editions", func(c *gin.Context) {
		v, err := s.Editions(c.Request.Context())
		Respond(c, v, err)
	})
	a.POST("/hourly-windows/seal", func(c *gin.Context) {
		var w hourly.Window
		if platform.Bind(c, &w) {
			v, err := s.Seal(c.Request.Context(), w)
			Respond(c, v, err)
		}
	})
	a.POST("/hourly-windows/replay", func(c *gin.Context) {
		var r hourly.Replay
		if platform.Bind(c, &r) {
			v, err := s.Replay(c.Request.Context(), r)
			Respond(c, v, err)
		}
	})
	a.POST("/hourly-windows/positions", func(c *gin.Context) {
		var p hourly.Positions
		if platform.Bind(c, &p) {
			v, err := s.Positions(c.Request.Context(), p)
			Respond(c, v, err)
		}
	})
	a.POST("/hourly-ranking-grants", func(c *gin.Context) {
		var g hourly.Grant
		if platform.Bind(c, &g) {
			v, err := s.Grant(c.Request.Context(), g)
			Respond(c, v, err)
		}
	})
}
