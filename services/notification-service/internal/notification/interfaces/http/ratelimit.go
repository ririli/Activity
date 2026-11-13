package http

import (
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

func admit(c *gin.Context, gate ratelimit.Gate) bool {
	if err := gate.Check(c.Request.Context(), platform.User(c)); err != nil {
		platform.Respond(c, nil, err)
		c.Abort()
		return false
	}
	return true
}
