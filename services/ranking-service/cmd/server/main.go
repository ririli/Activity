package main

import (
	"activity/pkg/platform"
	"activity/services/ranking-service/internal/ranking/application"
	store "activity/services/ranking-service/internal/ranking/infrastructure/mysql"
	httpapi "activity/services/ranking-service/internal/ranking/interfaces/http"
	"context"
)

func main() {
	db := platform.OpenDB()
	defer db.Close()
	r := platform.Redis()
	defer r.Close()
	repo := &store.Repository{DB: db, Redis: r}
	g := platform.Router(db, r)
	httpapi.Routes(g, application.Service{Repo: repo})
	platform.Run(g, func(ctx context.Context) {
		go platform.Dispatch(ctx, db)
		platform.Consume(ctx, "ranking.facts.v1", []string{"ParticipationInitialized.v1", "FormalDrawCommitted.v1", "RankingAwardGranted.v1"}, repo.Consume)
	})
}
