// Package testid supplies unique fixture IDs without a database reservation.
package testid

import (
	"context"
	"sync/atomic"
	"time"
)

var next atomic.Uint64

func init() { next.Store(uint64(time.Now().UnixNano())) }

func Next() uint64 { return next.Add(1) }

func Generate(context.Context) (uint64, error) { return Next(), nil }
