package application

import (
	domain "activity/services/activity-service/internal/assets/domain"
)

type GrantState struct {
	Reward     domain.RewardGrant
	Candidates []byte
}
type PendingGrant struct {
	ID         string
	Candidates []byte
}
type Change struct {
	Asset                  string
	Delta, Balance, Ledger int64
	Entry                  int
	Reason, Kind           string
}
type Grant struct {
	ID       uint64
	Round    int
	Prize    domain.Prize
	Source   string
	SourceID uint64
	No       int
	DrawID   uint64
}

// Repository is bound to one transaction and authenticated user/edition scope.
type Repository interface {
	ListPendingGrants() ([]PendingGrant, error)
	OwnedItemCount(itemID uint64) (int, error)
	FindPurchase(request string) (id, edition uint64, kind, product string, petals int64, err error)
	CountPurchaseQuota(product string) (int, error)
	InsertPurchase(request, kind, product string, cost, petals int64, version int) error
	InsertPurchaseQuota(product string) error
	LockReward(grantID string) (GrantState, error)
	MarkRewardSelected(grantID string, prize domain.Prize) error
	RewardSource(grantID string) (string, uint64, error)
	InitializeAccount(eventID uint64, w domain.Wallet) error
	FindRankingGrant(award uint64) (uint64, error)
	LockWallet() error
	InsertRankingGrant(grant, award uint64, item, name string, itemID uint64, quantity int64) error
	InsertPreviewTopUpFor(previewID, orderID uint64, request string, cost, petals int64, version int) error
	LoadWallet() (domain.Wallet, error)
	ApplyChange(Change) error
	SaveGrants([]Grant) error
}
