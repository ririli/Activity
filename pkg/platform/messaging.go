package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log/slog"
	"time"
)

type Event struct {
	ID      string          `json:"event_id"`
	Type    string          `json:"event_type"`
	Payload json.RawMessage `json:"payload"`
}

func channel() (*amqp.Connection, *amqp.Channel, error) {
	conn, e := amqp.DialConfig(Env("AMQP_URL", "amqp://activity:activity-local@localhost:15673/"), amqp.Config{Heartbeat: 10 * time.Second, Dial: amqp.DefaultDial(3 * time.Second)})
	if e != nil {
		return nil, nil, e
	}
	ch, e := conn.Channel()
	if e != nil {
		conn.Close()
		return nil, nil, e
	}
	e = ch.ExchangeDeclare("activity.events", "topic", true, false, false, false, nil)
	if e != nil {
		ch.Close()
		conn.Close()
		return nil, nil, e
	}
	return conn, ch, nil
}

// The outbox row lock is held until the broker confirms. A crash before marking
// PUBLISHED causes replay; consumers must commit their inbox before ACK.
func Dispatch(ctx context.Context, db *sql.DB, beforePublish ...func(context.Context, Event)) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for i := 0; i < 100 && ctx.Err() == nil; i++ {
				found, e := dispatchNext(ctx, db, publishEvent, beforePublish...)
				if e != nil {
					slog.Error("outbox retry", "error", e)
				}
				if !found {
					break
				}
			}
		}
	}
}
func dispatchNext(ctx context.Context, db *sql.DB, publish func(context.Context, Event) error, beforePublish ...func(context.Context, Event)) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	var ev Event
	var retries int
	e = tx.QueryRowContext(ctx, `SELECT CAST(event_id AS CHAR),event_type,payload,retry_count FROM outbox_event WHERE status='PENDING' AND next_retry_at<=UTC_TIMESTAMP(6) ORDER BY event_id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&ev.ID, &ev.Type, &ev.Payload, &retries)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	for _, hook := range beforePublish {
		hook(ctx, ev)
	}
	if e = publish(ctx, ev); e != nil {
		// Release the head row before scheduling its retry, including when the
		// publish deadline expired and database/sql already rolled back the tx.
		_ = tx.Rollback()
		retryCtx, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		delay := 5 * time.Second << min(retries, 6)
		delay = min(delay, 5*time.Minute)
		msg := []rune(e.Error())
		if len(msg) > 1024 {
			msg = msg[:1024]
		}
		_, retryErr := db.ExecContext(retryCtx, `UPDATE outbox_event SET retry_count=retry_count+1,next_retry_at=?,last_error=? WHERE event_id=? AND status='PENDING'`, time.Now().UTC().Add(delay), string(msg), ev.ID)
		if retryErr != nil {
			return false, fmt.Errorf("publish failed: %v; schedule retry: %w", e, retryErr)
		}
		return true, e
	}
	_, e = tx.ExecContext(ctx, `UPDATE outbox_event SET status='PUBLISHED',published_at=UTC_TIMESTAMP(6) WHERE event_id=?`, ev.ID)
	if e != nil {
		return false, e
	}
	return true, tx.Commit()
}

func publishEvent(ctx context.Context, ev Event) error {
	conn, ch, e := channel()
	if e != nil {
		return e
	}
	defer conn.Close()
	defer ch.Close()
	// amqp091 PublishWithContext does not interrupt socket writes itself.
	// Closing the connection makes both a blocked write and confirm wait stop.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if e = ch.Confirm(false); e != nil {
		return e
	}
	conf := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	ret := ch.NotifyReturn(make(chan amqp.Return, 1))
	if e = ch.PublishWithContext(ctx, "activity.events", ev.Type, true, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", MessageId: ev.ID, Body: JSON(ev)}); e != nil {
		return e
	}
	select {
	case c, ok := <-conf:
		if !ok || !c.Ack {
			return errors.New("publisher nack")
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-ret:
		return errors.New("unroutable event")
	default:
	}
	return nil
}
func Consume(ctx context.Context, name string, keys []string, handle func(context.Context, Event) error) {
	for ctx.Err() == nil {
		if e := consume(ctx, name, keys, handle); e != nil && ctx.Err() == nil {
			slog.Error("consumer reconnect", "queue", name, "error", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
func consume(ctx context.Context, name string, keys []string, handle func(context.Context, Event) error) error {
	conn, ch, e := channel()
	if e != nil {
		return e
	}
	defer conn.Close()
	defer ch.Close()
	_, e = ch.QueueDeclare(name+".dead", true, false, false, false, nil)
	if e != nil {
		return e
	}
	q, e := ch.QueueDeclare(name, true, false, false, false, amqp.Table{"x-dead-letter-exchange": "", "x-dead-letter-routing-key": name + ".dead"})
	if e != nil {
		return e
	}
	for _, k := range keys {
		if e = ch.QueueBind(q.Name, k, "activity.events", false, nil); e != nil {
			return e
		}
	}
	_ = ch.Qos(16, 0, false)
	deliveries, e := ch.Consume(q.Name, "", false, false, false, false, nil)
	if e != nil {
		return e
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("consumer disconnected")
			}
			var ev Event
			if e = json.Unmarshal(d.Body, &ev); e != nil {
				_ = d.Reject(false)
				continue
			}
			op, cancel := context.WithTimeout(ctx, 5*time.Second)
			e = handle(op, ev)
			cancel()
			if e != nil {
				var f *Fault
				if errors.As(e, &f) && f.Status == 409 {
					slog.Error("event quarantined", "id", ev.ID, "error", e)
					_ = d.Reject(false)
				} else {
					_ = d.Nack(false, true)
					return e
				}
			} else {
				_ = d.Ack(false)
			}
		}
	}
}
