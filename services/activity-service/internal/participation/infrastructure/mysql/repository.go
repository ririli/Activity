package mysql

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
	"context"
	"database/sql"
	"github.com/redis/go-redis/v9"
	"strconv"
)

// Repository owns only adapter concerns: the read paths and the background
// workflow's connection to the application service.
type Repository struct {
	DB     *sql.DB
	Redis  *redis.Client
	Notify chan []byte
	App    interface {
		Execute(context.Context, string, appstate.Command) (any, error)
		Internal(context.Context, string, string, string, string) (any, error)
		InitializeAccount(context.Context, platform.Event) error
		GrantRankingAward(context.Context, platform.Event) error
	}
}

func parseID(v string) (uint64, error) { return strconv.ParseUint(v, 10, 63) }
