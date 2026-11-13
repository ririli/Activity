package domain

import (
	"errors"
	"math"
)

const (
	Gold  = "GOLD"
	Petal = "PETAL"
)

var ErrInsufficient = errors.New("insufficient or overflowing asset")

type Wallet struct {
	Gold      int64
	Petals    int64
	LedgerSeq int64
	Version   int64
}

// Apply validates and applies one asset delta without touching persistence.
func (w *Wallet) Apply(asset string, delta int64) (int64, error) {
	if asset != Gold && asset != Petal {
		return 0, errors.New("invalid asset")
	}
	balance := w.Petals
	if asset == Gold {
		balance = w.Gold
	}
	if delta > 0 && balance > math.MaxInt64-delta || delta < 0 && (delta == math.MinInt64 || balance < -delta) {
		return 0, ErrInsufficient
	}
	balance += delta
	if asset == Gold {
		w.Gold = balance
	} else {
		w.Petals = balance
	}
	return balance, nil
}

func InitialWallet() Wallet { return Wallet{Gold: 10_000_000, Petals: 100_000_000, LedgerSeq: 2} }
