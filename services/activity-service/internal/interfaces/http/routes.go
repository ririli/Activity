package http

import (
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	app "activity/services/activity-service/internal/application"
	"activity/services/activity-service/internal/fault"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"strings"
)

type UseCases interface {
	Join(context.Context, app.JoinCommand) (any, error)
	Draw(context.Context, app.DrawCommand) (any, error)
	Purchase(context.Context, app.PurchaseCommand) (any, error)
	ClaimStage(context.Context, app.StageCommand) (any, error)
	OpenChest(context.Context, app.ChestCommand) (any, error)
	ConfirmChest(context.Context, app.ChestCommand) (any, error)
	SelectReward(context.Context, app.SelectRewardCommand) (any, error)
	NextRound(context.Context, app.RoundCommand) (any, error)
	Preview(context.Context, app.PreviewCommand) (any, error)
	ConfirmPreview(context.Context, app.PreviewCommand) (any, error)
	AbandonPreview(context.Context, app.PreviewCommand) (any, error)
	Read(context.Context, string, string, string, string, string) (any, error)
	Internal(context.Context, string, string, string, string) (any, error)
}

// Request is an HTTP DTO. Binding tags never enter application commands.
type Request struct {
	Edition   string `json:"edition_id" binding:"required"`
	Operation string `json:"operation_id"`
	Pool      string `json:"pool_code"`
	Count     int    `json:"count"`
	Product   string `json:"product_code"`
	Petals    int64  `json:"petals"`
	Stage     int    `json:"stage"`
	ID        string `json:"id"`
	Item      string `json:"item_id"`
	Round     int    `json:"from_round_no"`
}

func Respond(c *gin.Context, out any, err error) {
	var f *fault.Error
	if errors.As(err, &f) {
		status := 409
		if strings.HasPrefix(f.Code, "INVALID_") || f.Code == "UNKNOWN_ACTION" {
			status = 400
		}
		if f.Code == "DEPENDENCY_UNAVAILABLE" || f.Code == "OPERATION_RESULT_UNKNOWN" {
			status = 503
		}
		c.JSON(status, gin.H{"error": f.Code})
		return
	}
	if errors.Is(err, fault.ErrNotFound) {
		c.JSON(404, gin.H{"error": "NOT_FOUND"})
		return
	}
	platform.Respond(c, out, err)
}
func Routes(g *gin.Engine, s UseCases, limiters ...ratelimit.Limiter) {
	var limiter ratelimit.Limiter
	if len(limiters) > 0 {
		limiter = limiters[0]
	}
	queryLimit := ratelimit.Gate{Limiter: limiter, Service: "activity", Group: "query", Rule: ratelimit.Configure("RATE_LIMIT_ACTIVITY_QUERY", ratelimit.Rule{Rate: 10, Capacity: 30}), FailOpen: true}
	joinLimit := ratelimit.Gate{Limiter: limiter, Service: "activity", Group: "join", Rule: ratelimit.Configure("RATE_LIMIT_ACTIVITY_JOIN", ratelimit.Rule{Rate: 5, Capacity: 10}), FailOpen: false}
	operationLimit := ratelimit.Gate{Limiter: limiter, Service: "activity", Group: "operation", Rule: ratelimit.Configure("RATE_LIMIT_ACTIVITY_OPERATION", ratelimit.Rule{Rate: 2, Capacity: 4}), FailOpen: false}

	g.POST("/internal/activity/v1/:action", platform.Internal, func(c *gin.Context) {
		var v struct {
			Edition string `json:"edition_id"`
			User    string `json:"user_id"`
			After   string `json:"after"`
		}
		if !platform.Bind(c, &v) {
			return
		}
		out, err := s.Internal(c.Request.Context(), c.Param("action"), v.Edition, v.User, v.After)
		Respond(c, out, err)
	})
	a := g.Group("/api/activity/v1", platform.Auth)
	handlers := map[string]func(context.Context, app.Actor, Request) (any, error){
		"join": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.Join(ctx, app.JoinCommand{Actor: a})
		},
		"draw": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.Draw(ctx, app.DrawCommand{Actor: a, Operation: v.Operation, Pool: v.Pool, Count: v.Count})
		},
		"purchase": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.Purchase(ctx, app.PurchaseCommand{Actor: a, Operation: v.Operation, Product: v.Product, Petals: v.Petals})
		},
		"stage": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.ClaimStage(ctx, app.StageCommand{Actor: a, Stage: v.Stage})
		},
		"chest-open": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.OpenChest(ctx, app.ChestCommand{Actor: a})
		},
		"chest-confirm": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.ConfirmChest(ctx, app.ChestCommand{Actor: a, ID: v.ID})
		},
		"select": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.SelectReward(ctx, app.SelectRewardCommand{Actor: a, ID: v.ID, Item: v.Item})
		},
		"next-round": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.NextRound(ctx, app.RoundCommand{Actor: a, Round: v.Round})
		},
		"preview": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.Preview(ctx, app.PreviewCommand{Actor: a})
		},
		"preview-confirm": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.ConfirmPreview(ctx, app.PreviewCommand{Actor: a})
		},
		"preview-abandon": func(ctx context.Context, a app.Actor, v Request) (any, error) {
			return s.AbandonPreview(ctx, app.PreviewCommand{Actor: a})
		},
	}
	for action, handle := range handlers {
		a.POST("/"+action, func(c *gin.Context) {
			gate := operationLimit
			if action == "join" {
				gate = joinLimit
			}
			if !admit(c, gate) {
				return
			}
			var v Request
			if !platform.Bind(c, &v) {
				return
			}
			actor := app.Actor{User: platform.User(c), Edition: v.Edition, DisplayName: c.MustGet("principal").(platform.Principal).Nickname}
			out, err := handle(c.Request.Context(), actor, v)
			Respond(c, out, err)
		})
	}
	a.GET("/records/:kind", func(c *gin.Context) {
		if !admit(c, queryLimit) {
			return
		}
		out, err := s.Read(c.Request.Context(), platform.User(c), c.Query("edition_id"), c.Param("kind"), c.Query("before"), c.Query("status"))
		Respond(c, out, err)
	})
	a.POST("/tasks/claim", func(c *gin.Context) { c.JSON(409, gin.H{"error": "TASKS_NOT_OPEN"}) })
}
