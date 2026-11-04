package main

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appeffects"
	"activity/services/activity-service/internal/applock"
	activityapp "activity/services/activity-service/internal/application"
	participationapp "activity/services/activity-service/internal/participation/application"
	store "activity/services/activity-service/internal/participation/infrastructure/mysql"
	rankingclient "activity/services/activity-service/internal/participation/infrastructure/ranking"
	httpapi "activity/services/activity-service/internal/participation/interfaces/http"
	"activity/services/activity-service/internal/uow"
	"context"
)

type legacyRepo struct{ repo *store.Repository }

func (l legacyRepo) Read(ctx context.Context, user, edition, kind, before, status string) (any, error) {
	return l.repo.Read(ctx, user, edition, kind, before, status)
}
func (l legacyRepo) Internal(ctx context.Context, action, edition, user, after string) (any, error) {
	return l.repo.Internal(ctx, action, edition, user, after)
}

func main() {
	db := platform.OpenDB()
	defer db.Close()
	r := platform.Redis()
	defer r.Close()
	repo := &store.Repository{DB: db, Redis: r, Notify: make(chan []byte, 1024)}
	app := activityapp.Service{
		UoW:     uow.MySQL{DB: db},
		Lock:    applock.Redis{Client: r},
		Effects: appeffects.Effects{Notify: repo.Notify},
		Legacy:  legacyRepo{repo: repo},
	}
	repo.App = &app
	tasks := store.TaskStore{DB: db}
	worker := participationapp.SettlementWorker{
		Store:     tasks,
		Exec:      &app,
		Ranking:   rankingclient.Client{URL: platform.Env("RANKING_URL", "http://localhost:8083")},
		Bootstrap: repo,
		Batch:     tasks.Batch(),
	}
	g := platform.Router(db, r)
	httpapi.Routes(g, app)
	platform.Run(g, func(ctx context.Context) {
		go platform.Dispatch(ctx, db, store.TryRanking)
		go worker.Run(ctx)
		go repo.Publish(ctx)
		go platform.Consume(ctx, "activity.awards.v1", []string{"RankingAwardRequested.v1"}, app.GrantRankingAward)
		platform.Consume(ctx, "activity.accounts.v1", []string{"AccountCreated.v1"}, app.InitializeAccount)
	})
}
