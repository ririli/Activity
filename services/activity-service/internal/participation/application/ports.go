package application

import (
	domain "activity/services/activity-service/internal/participation/domain"
)

// Repository is bound to one transaction and authenticated user/edition scope.
type Repository interface {
	ListRounds() ([]int, error)
	LockRoundForSettlement(round int) (flowers, mask int, rules []byte, err error)
	ConfirmAllChests() error
	RecomputeCompletedChests() error
	FindStageClaim(stage int) (uint64, error)
	InsertStageClaim(claimID uint64, stage int, grant uint64, source string) error
	UpdateRoundMask(mask int) error
	FindOpenChest() (id, grant uint64, raw []byte, err error)
	UsedChestCodes() ([]string, error)
	InsertChest(id uint64, chestNo int, code, kind string, snapshot []byte, grant uint64) error
	LockChest(id string) (status, reward string, round int, err error)
	ConfirmChest(id string) error
	IncrementCompletedChests() error
	PendingStageChoices(round int) (int, error)
	CloseRound(round int) error
	InsertRound(round, version int) error
	IncrementParticipationRound() error
	LockEditionForClose() (domain.Edition, error)
	MarkEditionEnded() error
	Gate() (domain.Edition, error)
	Exists() (bool, error)
	Initialize(int, int64) error
	LoadProgress() (domain.Progress, error)
	Rule(int) ([]byte, []byte, error)
	SaveDrawProgress(domain.Progress) error
}
