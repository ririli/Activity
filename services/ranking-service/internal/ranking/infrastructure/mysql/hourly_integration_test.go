package mysql

import (
	"activity/pkg/hourly"
	"activity/pkg/platform"
	app "activity/services/ranking-service/internal/ranking/application"
	"activity/services/ranking-service/internal/ranking/domain"
	cache "activity/services/ranking-service/internal/ranking/infrastructure/redis"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	rdb "github.com/redis/go-redis/v9"
)

type hourlyTestActivity struct {
	edition     hourly.Edition
	events      map[int64][]platform.Event
	receipts    map[string]string
	loseReceipt bool
}

func (a *hourlyTestActivity) Editions(context.Context) ([]hourly.Edition, error) {
	return []hourly.Edition{a.edition}, nil
}

func (a *hourlyTestActivity) Seal(ctx context.Context, w hourly.Window) (hourly.Watermark, error) {
	var mark hourly.Watermark
	for shard := range mark {
		positions, err := a.Positions(ctx, hourly.Positions{Window: w, Shard: shard})
		if err != nil {
			return mark, err
		}
		mark[shard] = positions
	}
	return mark, nil
}

func (a *hourlyTestActivity) Positions(_ context.Context, req hourly.Positions) ([hourly.Lanes]int64, error) {
	var mark [hourly.Lanes]int64
	for _, ev := range a.events[req.Cutoff] {
		var f domain.Fact
		if err := json.Unmarshal(ev.Payload, &f); err != nil {
			return mark, err
		}
		if p := f.Hourly; p.Shard == req.Shard {
			mark[p.Lane] = max(mark[p.Lane], p.Seq)
		}
	}
	return mark, nil
}

func (a *hourlyTestActivity) Replay(_ context.Context, req hourly.Replay) ([]platform.Event, error) {
	var page []platform.Event
	for _, ev := range a.events[req.Cutoff] {
		var f domain.Fact
		if err := json.Unmarshal(ev.Payload, &f); err != nil {
			return nil, err
		}
		p := f.Hourly
		if p.Shard == req.Shard && p.Lane == req.Lane && p.Seq > req.After && p.Seq <= req.Through {
			page = append(page, ev)
		}
	}
	return page, nil
}

func (a *hourlyTestActivity) Grant(_ context.Context, g hourly.Grant) (hourly.Receipt, error) {
	key := g.Settlement + "/" + g.User
	if a.receipts[key] == "" {
		a.receipts[key] = fmt.Sprint(platform.ID())
	}
	if a.loseReceipt {
		a.loseReceipt = false
		return hourly.Receipt{}, errors.New("response lost after commit")
	}
	return hourly.Receipt{GrantID: a.receipts[key]}, nil
}

func TestHourlyBarrierRecovery(t *testing.T) {
	dsn, addr := os.Getenv("RANKING_TEST_MYSQL_DSN"), os.Getenv("RANKING_TEST_REDIS_ADDR")
	physical := os.Getenv("RANKING_TEST_REDIS_SHARD_ADDRS")
	if dsn == "" || (addr == "" && physical == "") || os.Getenv("RANKING_TEST_PREPARED_SCHEMA") != "1" {
		t.Skip("requires disposable initialized ranking MySQL and Redis")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var clients [hourly.Shards]*rdb.Client
	addresses := []string{addr, addr, addr, addr}
	password := ""
	if physical != "" {
		addresses = strings.Split(physical, ",")
		password = os.Getenv("RANKING_TEST_REDIS_PASSWORD")
		if len(addresses) != len(clients) {
			t.Fatal("RANKING_TEST_REDIS_SHARD_ADDRS must specify 4 addresses")
		}
	}
	for i := range clients {
		clients[i] = rdb.NewClient(&rdb.Options{Addr: addresses[i], Password: password})
		defer clients[i].Close()
	}
	store := HourlyStore{DB: db}
	last := time.Now().UTC().Truncate(time.Hour)
	first := last.Add(-time.Hour)
	start, end := first.Add(-time.Minute), last.Add(time.Hour)
	edition := hourly.Edition{ID: fmt.Sprint(platform.ID()), Start: start, End: end}
	source := &hourlyTestActivity{edition: edition, events: make(map[int64][]platform.Event), receipts: make(map[string]string), loseReceipt: true}
	projection := cache.HourlyCache{Clients: clients, Namespace: "hourly-worker-test", LoadBounds: func(context.Context, string) (cache.Bounds, error) {
		return cache.Bounds{Min: start.Unix(), End: end.Unix()}, nil
	}}
	worker := app.HourlyWorker{Store: store, Activity: source, Projection: projection, Wake: make(chan struct{}, 1)}
	var schedulers sync.WaitGroup
	for range 4 {
		schedulers.Go(func() {
			if _, err := store.ScheduleHourly(ctx, []hourly.Edition{edition}); err != nil {
				t.Error(err)
			}
		})
	}
	schedulers.Wait()
	var scheduled int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hourly_settlement WHERE edition_id=?`, edition.ID).Scan(&scheduled); err != nil || scheduled != 2 {
		t.Fatalf("scheduled %d cutoffs: %v", scheduled, err)
	}
	var barrier platform.Event
	if broker := os.Getenv("RANKING_TEST_AMQP_URL"); broker != "" {
		t.Setenv("AMQP_URL", broker)
		mq, stop := context.WithCancel(ctx)
		defer stop()
		var background sync.WaitGroup
		defer func() { stop(); background.Wait() }()
		background.Go(func() {
			platform.Consume(mq, "ranking.hourly.integration."+edition.ID, []string{hourly.BarrierType}, worker.Barrier)
		})
		background.Go(func() { platform.Dispatch(mq, db) })
		timer := time.NewTicker(50 * time.Millisecond)
		defer timer.Stop()
		for {
			var received int
			if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hourly_settlement WHERE edition_id=? AND barrier_received_at IS NOT NULL`, edition.ID).Scan(&received); err != nil {
				t.Fatal(err)
			}
			if received == 2 {
				break
			}
			select {
			case <-ctx.Done():
				t.Fatal("barrier did not reach the RabbitMQ consumer")
			case <-timer.C:
			}
		}
	}
	for _, at := range []time.Time{last, first} {
		err := db.QueryRowContext(ctx, `SELECT CAST(o.event_id AS CHAR),o.event_type,o.payload FROM outbox_event o
			JOIN hourly_settlement h ON h.settlement_id=o.event_id WHERE h.edition_id=? AND h.cutoff_sec=?`, edition.ID, at.Unix()).Scan(&barrier.ID, &barrier.Type, &barrier.Payload)
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if err := worker.Barrier(ctx, barrier); err != nil {
				t.Fatal(err)
			}
		}
	}
	// The Barrier has arrived, but the first window's facts have not been delivered.
	var counts [hourly.Shards][hourly.Lanes]int64
	for user := uint64(1); user <= 104; user++ {
		shard, lane := hourly.Shard(user), hourly.Lane(user)
		counts[shard][lane]++
		p := hourly.Position{Cutoff: first.Unix(), Shard: shard, Lane: lane, Seq: counts[shard][lane]}
		f := domain.Fact{Edition: edition.ID, User: fmt.Sprint(user), Created: fmt.Sprint(start.Unix()), Hourly: &p}
		source.events[first.Unix()] = append(source.events[first.Unix()], platform.Event{Type: "ParticipationInitialized.v1", Payload: platform.JSON(f)})
	}
	changed := domain.Fact{Edition: edition.ID, User: "104", Draw: "501", Total: "10", Reached: fmt.Sprint(first.Unix()),
		Hourly: &hourly.Position{Cutoff: last.Unix(), Shard: hourly.Shard(104), Lane: hourly.Lane(104), Seq: 1}}
	source.events[last.Unix()] = []platform.Event{{Type: "FormalDrawCommitted.v1", Payload: platform.JSON(changed)}}
	if err := worker.Project(ctx, changed); err != nil {
		t.Fatal(err)
	}
	// Replication preserves the old primary's metadata when a replica is promoted.
	// Even with the correct member count, the window must be replayed for the new process.
	shard := changed.Hourly.Shard
	metaKey := fmt.Sprintf("ranking:hourly-worker-test:{%s:%d}:hourly:%d:meta", edition.ID, shard, last.Unix())
	if err := clients[shard].HSet(ctx, metaKey, "run_id", "previousprimary").Err(); err != nil {
		t.Fatal(err)
	}
	if positions, err := projection.Positions(ctx, hourly.Window{Edition: edition.ID, Cutoff: last.Unix()}, shard); err != nil || positions[changed.Hourly.Lane] != 0 {
		t.Fatalf("changed primary must invalidate the old cursor: %v, %v", positions, err)
	}
	claimed, err := store.ClaimHourly(ctx)
	if err != nil || claimed == nil || claimed.Cutoff != first.Unix() {
		t.Fatalf("claim before takeover: %+v %v", claimed, err)
	}
	if blocked, err := store.ClaimHourly(ctx); err != nil || blocked != nil {
		t.Fatalf("second worker bypassed active lease or predecessor: %+v %v", blocked, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE hourly_settlement SET lease_until=DATE_SUB(UTC_TIMESTAMP(6),INTERVAL 1 SECOND) WHERE settlement_id=?`, claimed.Settlement); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := worker.SettleNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.FreezeHourly(ctx, *claimed, nil); err == nil {
		t.Fatal("stale owner was allowed to rewrite a frozen result")
	}
	var count, rank int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hourly_award WHERE edition_id=? AND cutoff_sec=?`, edition.ID, first.Unix()).Scan(&count); err != nil || count != 100 {
		t.Fatalf("first window count %d: %v", count, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT final_rank FROM hourly_award WHERE edition_id=? AND cutoff_sec=? AND user_id=104`, edition.ID, last.Unix()).Scan(&rank); err != nil || rank != 1 {
		t.Fatalf("later window rank %d: %v", rank, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hourly_award WHERE edition_id=? AND cutoff_sec=? AND user_id=100`, edition.ID, last.Unix()).Scan(&count); err != nil || count != 0 {
		t.Fatalf("former cutoff user was not displaced: %d %v", count, err)
	}
	if found, err := worker.DeliverNext(ctx); !found || err == nil {
		t.Fatalf("lost response: found=%v error=%v", found, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE hourly_award SET next_retry_at=UTC_TIMESTAMP(6) WHERE edition_id=? AND retry_count=1`, edition.ID); err != nil {
		t.Fatal(err)
	}
	for range 200 {
		if found, err := worker.DeliverNext(ctx); !found || err != nil {
			t.Fatalf("grant resume: found=%v error=%v", found, err)
		}
	}
	if len(source.receipts) != 200 {
		t.Fatalf("expected 200 distinct receipts, got %d", len(source.receipts))
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hourly_settlement WHERE edition_id=? AND status='COMPLETED'`, edition.ID).Scan(&count); err != nil || count != 2 {
		t.Fatalf("completed %d windows: %v", count, err)
	}
}
