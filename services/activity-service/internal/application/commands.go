package application

type Actor struct{ User, Edition, DisplayName string }
type JoinCommand struct{ Actor }
type DrawCommand struct {
	Actor
	Operation, Pool string
	Count           int
}
type PurchaseCommand struct {
	Actor
	Operation, Product string
	Petals             int64
}
type SelectRewardCommand struct {
	Actor
	ID, Item string
}
type StageCommand struct {
	Actor
	Stage int
}
type ChestCommand struct {
	Actor
	ID string
}
type RoundCommand struct {
	Actor
	Round int
}
type PreviewCommand struct{ Actor }
type RankingAwardCommand struct {
	User, Edition, Award, Item, Name string
	Quantity                         int64
}
type AccountCommand struct{ User, EventID string }
