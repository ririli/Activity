package http

import (
	"activity/pkg/platform"
	"activity/services/identity-service/internal/identity/application"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

const cookie = "activity_refresh"
const path = "/api/identity/v1/auth"

func Routes(g *gin.Engine, s *application.Service) {
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
		var v application.Credentials
		if !platform.Bind(c, &v) {
			return
		}
		t, e := s.Register(c.Request.Context(), v, c.ClientIP())
		set(c, t, e)
	})
	auth.POST("/login", func(c *gin.Context) {
		var v application.Credentials
		if !platform.Bind(c, &v) {
			return
		}
		t, e := s.Login(c.Request.Context(), v)
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
		if e == nil {
			c.SetCookie(cookie, "", -1, path, "", os.Getenv("COOKIE_SECURE") != "false", true)
		}
		platform.Respond(c, gin.H{"status": "ok"}, e)
	})
	g.GET("/api/identity/v1/me", func(c *gin.Context) {
		p, e := s.Introspect(c.Request.Context(), strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
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
	internal.POST("/profiles", func(c *gin.Context) {
		var v struct {
			IDs []string `json:"ids" binding:"required,max=100"`
		}
		if !platform.Bind(c, &v) {
			return
		}
		p, e := s.Repo.Profiles(c.Request.Context(), v.IDs)
		platform.Respond(c, p, e)
	})
}
