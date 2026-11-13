package infrastructure

import (
	"activity/services/notification-service/internal/notification/application"
	"testing"
)

func TestHubConfig(t *testing.T) {
	names := []string{"NOTIFICATION_HUB_SHARDS", "NOTIFICATION_MAX_CONNECTIONS", "NOTIFICATION_SESSION_LIMIT", "NOTIFICATION_SHARD_QUEUE", "NOTIFICATION_CONNECTION_QUEUE"}
	for _, name := range names {
		t.Setenv(name, "1")
	}
	c, err := HubConfig()
	if err != nil || c != (application.Config{Shards: 1, MaxConnections: 1, SessionLimit: 1, ShardQueue: 1, ConnectionQueue: 1}) {
		t.Fatalf("%+v %v", c, err)
	}
	for _, name := range names {
		for _, raw := range []string{"", "0", "-1", "abc", "1.5", "99999999999999999999999999"} {
			t.Run(name+"/"+raw, func(t *testing.T) {
				t.Setenv(name, raw)
				if _, err := HubConfig(); err == nil {
					t.Fatal("accepted invalid environment")
				}
			})
		}
	}
}
