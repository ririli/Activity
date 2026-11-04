package http

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/application"
	"github.com/gin-gonic/gin"
)

func Routes(g *gin.Engine, s application.Service) {
	g.POST("/internal/ranking/v1/projection", platform.Internal, func(c *gin.Context) {
		var ev platform.Event
		if !platform.Bind(c, &ev) {
			return
		}
		e := s.Repo.CacheUpdate(c.Request.Context(), ev)
		platform.Respond(c, gin.H{"status": "ok"}, e)
	})
	g.GET("/api/ranking/v1/board", platform.Auth, func(c *gin.Context) {
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
