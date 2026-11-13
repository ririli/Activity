package infrastructure

import (
	"activity/services/notification-service/internal/notification/application"
	"fmt"
	"os"
	"strconv"
)

func HubConfig() (application.Config, error) {
	c := application.DefaultConfig()
	for _, p := range []struct {
		name   string
		target *int
	}{
		{"NOTIFICATION_HUB_SHARDS", &c.Shards},
		{"NOTIFICATION_MAX_CONNECTIONS", &c.MaxConnections},
		{"NOTIFICATION_SESSION_LIMIT", &c.SessionLimit},
		{"NOTIFICATION_SHARD_QUEUE", &c.ShardQueue},
		{"NOTIFICATION_CONNECTION_QUEUE", &c.ConnectionQueue},
	} {
		if raw, exists := os.LookupEnv(p.name); exists {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				return c, fmt.Errorf("%s must be a positive integer", p.name)
			}
			*p.target = n
		}
	}
	return c, c.Validate()
}
