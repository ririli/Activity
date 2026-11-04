package domain

// SettlementResult is the internal contract returned to activity. Status is one
// of PENDING, RECONCILING, FROZEN, AWARDING, COMPLETED or FAILED.
type SettlementResult struct {
	Status        string `json:"status"`
	PendingAwards int64  `json:"pending_awards"`
}

const (
	SettlementPending     = "PENDING"
	SettlementReconciling = "RECONCILING"
	SettlementFrozen      = "FROZEN"
	SettlementAwarding    = "AWARDING"
	SettlementCompleted   = "COMPLETED"
	SettlementFailed      = "FAILED"
)
