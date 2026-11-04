package http

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/participation/application"
	"github.com/gin-gonic/gin"
)

func Routes(g *gin.Engine, s application.Service) {
	g.POST("/internal/activity/v1/:action", platform.Internal, func(c *gin.Context) {
		var v struct {
			Edition string `json:"edition_id"`
			User    string `json:"user_id"`
			After   string `json:"after"`
		}
		if !platform.Bind(c, &v) {
			return
		}
		out, e := s.Repo.Internal(c.Request.Context(), c.Param("action"), v.Edition, v.User, v.After)
		platform.Respond(c, out, e)
	})
	a := g.Group("/api/activity/v1", platform.Auth)
	for _, action := range []string{"join", "draw", "purchase", "stage", "chest-open", "chest-confirm", "select", "next-round", "preview", "preview-confirm", "preview-abandon"} {
		action := action
		a.POST("/"+action, func(c *gin.Context) {
			var cmd application.Command
			if !platform.Bind(c, &cmd) {
				return
			}
			cmd.Action = action
			cmd.DisplayName = c.MustGet("principal").(platform.Principal).Nickname
			v, e := s.Execute(c.Request.Context(), platform.User(c), cmd)
			platform.Respond(c, v, e)
		})
	}
	a.GET("/records/:kind", func(c *gin.Context) {
		v, e := s.Repo.Read(c.Request.Context(), platform.User(c), c.Query("edition_id"), c.Param("kind"), c.Query("before"))
		platform.Respond(c, v, e)
	})
	a.POST("/tasks/claim", func(c *gin.Context) { c.JSON(409, gin.H{"error": "TASKS_NOT_OPEN"}) })
}
