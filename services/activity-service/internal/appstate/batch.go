package appstate

import (
	"activity/services/activity-service/internal/uow"
	"context"
	"errors"
	"strings"
)

const (
	batchMaxRows         = 100
	batchMaxPlaceholders = 60000
)

func BatchInsert(ctx context.Context, tx uow.Tx, table string, columns []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	if len(columns) == 0 || len(table) == 0 {
		return errors.New("invalid batch specification")
	}
	rowLimit := batchMaxRows
	if n := batchMaxPlaceholders / len(columns); n < rowLimit {
		rowLimit = n
	}
	if rowLimit < 1 {
		return errors.New("invalid batch column count")
	}
	mark := "(" + strings.TrimSuffix(strings.Repeat("?,", len(columns)), ",") + ")"
	for start := 0; start < len(rows); start += rowLimit {
		end := start + rowLimit
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[start:end]
		values := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*len(columns))
		for i, row := range chunk {
			if len(row) != len(columns) {
				return errors.New("invalid batch row width")
			}
			values[i] = mark
			args = append(args, row...)
		}
		query := "INSERT INTO " + table + "(" + strings.Join(columns, ",") + ") VALUES " + strings.Join(values, ",")
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}
