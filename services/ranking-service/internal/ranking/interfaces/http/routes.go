package http

import (
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	"activity/services/ranking-service/internal/ranking/application"
	"github.com/gin-gonic/gin"
)

func Routes(g *gin.Engine, s application.Service, limiters ...ratelimit.Limiter) {
	var limiter ratelimit.Limiter
	if len(limiters) > 0 {
		limiter = limiters[0]
	}
	boardLimit := ratelimit.Gate{Limiter: limiter, Service: "ranking", Group: "board", Rule: ratelimit.Configure("RATE_LIMIT_RANKING_BOARD", ratelimit.Rule{Rate: 2, Capacity: 5}), FailOpen: true}

	g.GET("/api/ranking/v1/board", platform.Auth, func(c *gin.Context) {
		if !admit(c, boardLimit) {
			return
		}
		v, e := s.Board(c.Request.Context(), c.Query("edition_id"), platform.User(c))
		platform.Respond(c, v, e)
	})
	g.POST("/internal/ranking/v1/settle", platform.Internal, func(c *gin.Context) {
		var v struct {
			Edition string `json:"edition_id" binding:"required"`
		}
		if !platform.Bind(c, &v) {
			return
		}
		result, e := s.Advance(c.Request.Context(), v.Edition)
		platform.Respond(c, result, e)
	})
}
