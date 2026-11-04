package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type Message struct {
	Version  int       `json:"schema_version"`
	ID       string    `json:"event_id"`
	Type     string    `json:"event_type"`
	Occurred time.Time `json:"occurred_at"`
	Expires  time.Time `json:"expires_at"`
	Edition  string    `json:"edition_id"`
	Source   string    `json:"source"`
	Name     string    `json:"display_name"`
	ItemID   uint64    `json:"item_id"`
	ItemName string    `json:"item_name"`
	Quantity int64     `json:"quantity"`
}

func Parse(raw []byte, now time.Time) (Message, error) {
	var m Message
	if len(raw) > 4096 {
		return m, errors.New("message too large")
	}
	e := json.Unmarshal(raw, &m)
	if e != nil {
		return m, e
	}
	if m.Version != 1 || m.Type != "ring_awarded" || len(m.ID) == 0 || len(m.ID) > 160 || strings.ContainsAny(m.ID, "\r\n") || !utf8.ValidString(m.Name) || len([]rune(m.Name)) > 64 || m.Name == "" || len([]rune(m.ItemName)) > 128 || m.ItemName == "" || m.Quantity <= 0 || m.ItemID == 0 || !m.Expires.After(now) || m.Expires.Sub(m.Occurred) > time.Minute || m.Occurred.After(now.Add(5*time.Second)) {
		return m, errors.New("invalid notification")
	}
	if m.Source != "normal_draw" && m.Source != "normal_draw_selection" && m.Source != "pay_later_selection" {
		return m, errors.New("unsupported notification source")
	}
	return m, nil
}
