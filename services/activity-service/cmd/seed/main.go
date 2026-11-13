package main

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/application"
	store "activity/services/activity-service/internal/infrastructure/mysql"
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"time"
)

func main() {
	db := platform.OpenDB()
	defer db.Close()
	start, e := time.Parse(time.RFC3339, os.Getenv("EDITION_START"))
	platform.Must(e)
	end, e := time.Parse(time.RFC3339, os.Getenv("EDITION_END"))
	platform.Must(e)
	edition, e := platform.UID(os.Getenv("EDITION_ID"))
	platform.Must(e)
	raw := platform.JSON(application.DefaultRules())
	var obj any
	platform.Must(json.Unmarshal(raw, &obj))
	canonical := platform.JSON(obj)
	hash := sha256.Sum256(canonical)
	tx, e := db.BeginTx(context.Background(), nil)
	platform.Must(e)
	defer tx.Rollback()
	hourlyEnabled := os.Getenv("EDITION_HOURLY_RANKING") == "true"
	_, e = tx.Exec(`INSERT INTO activity_edition(edition_id,activity_code,name,starts_at,ends_at,status,active_rule_version,hourly_ranking_enabled) VALUES(?,'lottery','花朵抽奖',?,?,'ACTIVE',1,?)`, edition, start.UTC(), end.UTC(), hourlyEnabled)
	platform.Must(e)
	if hourlyEnabled {
		platform.Must(store.PrepareHourlyWindows(context.Background(), tx, edition, start, end))
	}
	_, e = tx.Exec(`INSERT INTO activity_rule_version(edition_id,version,schema_version,rule_json,checksum) VALUES(?,1,1,?,?)`, edition, canonical, hash[:])
	platform.Must(e)
	platform.Must(tx.Commit())
}
