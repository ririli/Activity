package main

import (
	"activity/pkg/platform"
	"activity/pkg/ratelimit"
	app "activity/services/activity-service/internal/application"
	effects "activity/services/activity-service/internal/infrastructure/effects"
	store "activity/services/activity-service/internal/infrastructure/mysql"
	ranking "activity/services/activity-service/internal/infrastructure/ranking"
	locks "activity/services/activity-service/internal/infrastructure/redis"
	httpapi "activity/services/activity-service/internal/interfaces/http"
	"activity/services/activity-service/internal/interfaces/messaging"
	"activity/services/activity-service/internal/interfaces/worker"
	lottery "activity/services/activity-service/internal/lottery/domain"
	"context"
	"time"
)

func main() {
	db := platform.OpenDB()
	defer db.Close()
	redis := platform.Redis()
	defer redis.Close()
	notify := make(chan []byte, 1024)
	service := app.Service{UoW: store.UnitOfWork{DB: db}, Lock: locks.Redis{Client: redis}, Effects: effects.Effects{Notify: notify}, Queries: &store.Queries{DB: db}, Runtime: app.Runtime{NewID: platform.ID, Now: time.Now, Random: lottery.Random}}
	tasks := store.TaskStore{DB: db}
	settlement := worker.Settlement{Workflow: app.SettlementWorker{Store: tasks, Exec: service, Ranking: ranking.Client{URL: platform.Env("RANKING_URL", "http://localhost:8083")}, Bootstrap: service, Batch: tasks.Batch(), Disabled: platform.Env("SETTLEMENT_ENABLED", "true") == "false"}}
	consumers := messaging.Consumers{App: service}
	publisher := effects.Publisher{Redis: redis, Notify: notify}
	limiter := ratelimit.NewRedis()
	defer limiter.Close()
	g := platform.Router(db, redis)
	httpapi.Routes(g, service, limiter)
	httpapi.HourlyRoutes(g, app.HourlyService{Queries: &store.Queries{DB: db}, UoW: service.UoW, NewID: platform.ID})
	platform.Run(g, func(ctx context.Context) {
		go platform.Dispatch(ctx, db)
		go settlement.Run(ctx)
		go publisher.Publish(ctx)
		go platform.Consume(ctx, "activity.awards.v1", []string{"RankingAwardRequested.v1"}, consumers.GrantRankingAward)
		platform.Consume(ctx, "activity.accounts.v1", []string{"AccountCreated.v1"}, consumers.InitializeAccount)
	})
}
