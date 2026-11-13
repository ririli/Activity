package http

import (
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	"activity/services/identity-service/internal/identity/application"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

const cookie = "activity_refresh"
const path = "/api/identity/v1/auth"

type credentials struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}

func (v credentials) command() application.Credentials {
	return application.Credentials{Account: v.Account, Password: v.Password, Nickname: v.Nickname}
}

func Routes(g *gin.Engine, s *application.Service, limiters ...ratelimit.Limiter) {
	var limiter ratelimit.Limiter
	if len(limiters) > 0 {
		limiter = limiters[0]
	}
	meLimit := ratelimit.Gate{Limiter: limiter, Service: "identity", Group: "me", Rule: ratelimit.Configure("RATE_LIMIT_IDENTITY_ME", ratelimit.Rule{Rate: 5, Capacity: 10}), FailOpen: true}

	auth := g.Group(path)
	auth.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store") })
	set := func(c *gin.Context, t application.Tokens, e error) {
		if e == nil {
			c.SetSameSite(http.SameSiteStrictMode)
			c.SetCookie(cookie, t.Refresh, 7*86400, path, "", os.Getenv("COOKIE_SECURE") != "false", true)
		}
		platform.Respond(c, t, e)
	}
	auth.POST("/register", func(c *gin.Context) {
		var v credentials
		if !platform.Bind(c, &v) {
			return
		}
		t, e := s.Register(c.Request.Context(), v.command(), c.ClientIP())
		set(c, t, e)
	})
	auth.POST("/login", func(c *gin.Context) {
		var v credentials
		if !platform.Bind(c, &v) {
			return
		}
		t, e := s.Login(c.Request.Context(), v.command())
		set(c, t, e)
	})
	csrf := func(c *gin.Context) {
		if c.GetHeader("Origin") != os.Getenv("ALLOWED_ORIGIN") || c.GetHeader("X-Activity-CSRF") != "1" || os.Getenv("ALLOWED_ORIGIN") == "" {
			c.AbortWithStatus(403)
		}
	}
	auth.POST("/refresh", csrf, func(c *gin.Context) {
		raw, _ := c.Cookie(cookie)
		t, e := s.Renew(c.Request.Context(), raw)
		set(c, t, e)
	})
	auth.POST("/logout", csrf, func(c *gin.Context) {
		raw, _ := c.Cookie(cookie)
		e := s.Logout(c.Request.Context(), raw)
		// Clear the cookie on success, idempotent no-cookie logout, and invalid
		// signed cookies. An invalid cookie still returns 401.
		c.SetCookie(cookie, "", -1, path, "", os.Getenv("COOKIE_SECURE") != "false", true)
		platform.Respond(c, gin.H{"status": "ok"}, e)
	})
	g.GET("/api/identity/v1/me", func(c *gin.Context) {
		p, e := s.Introspect(c.Request.Context(), strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if e == nil {
			e = meLimit.Check(c.Request.Context(), p.UserID)
		}
		platform.Respond(c, p, e)
	})
	internal := g.Group("/internal/identity/v1", platform.Internal)
	internal.POST("/introspect", func(c *gin.Context) {
		var v struct {
			Token string `json:"token" binding:"required"`
		}
		if !platform.Bind(c, &v) {
			return
		}
		p, e := s.Introspect(c.Request.Context(), v.Token)
		platform.Respond(c, p, e)
	})
}
