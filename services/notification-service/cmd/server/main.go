package main

import (
	"activity/pkg/platform"
	"activity/services/notification-service/internal/notification/application"
	"activity/services/notification-service/internal/notification/infrastructure"
	httpapi "activity/services/notification-service/internal/notification/interfaces/http"
	"context"
)

func main() {
	r := platform.Redis()
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := application.New(ctx)
	g := platform.Router(nil, r)
	httpapi.Routes(g, h)
	platform.Run(g, func(ctx context.Context) { defer cancel(); infrastructure.Subscribe(ctx, r, h) })
}
