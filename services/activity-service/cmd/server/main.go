package main

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/participation/application"
	store "activity/services/activity-service/internal/participation/infrastructure/mysql"
	httpapi "activity/services/activity-service/internal/participation/interfaces/http"
	"context"
)

func main() {
	db := platform.OpenDB()
	defer db.Close()
	r := platform.Redis()
	defer r.Close()
	repo := &store.Repository{DB: db, Redis: r, Notify: make(chan []byte, 1024)}
	s := application.Service{Repo: repo}
	g := platform.Router(db, r)
	httpapi.Routes(g, s)
	platform.Run(g, func(ctx context.Context) {
		go platform.Dispatch(ctx, db)
		go repo.Maintenance(ctx)
		go repo.Publish(ctx)
		go platform.Consume(ctx, "activity.awards.v1", []string{"RankingAwardRequested.v1"}, repo.GrantAward)
		platform.Consume(ctx, "activity.accounts.v1", []string{"AccountCreated.v1"}, s.Consume)
	})
}
