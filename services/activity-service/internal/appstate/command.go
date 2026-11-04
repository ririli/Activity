package appstate

// Command is the transport-neutral input shared by the activity use cases.
type Command struct {
	Edition     string `json:"edition_id" binding:"required"`
	Operation   string `json:"operation_id"`
	Pool        string `json:"pool_code"`
	Count       int    `json:"count"`
	Product     string `json:"product_code"`
	Petals      int64  `json:"petals"`
	Stage       int    `json:"stage"`
	ID          string `json:"id"`
	Item        string `json:"item_id"`
	Round       int    `json:"from_round_no"`
	DisplayName string `json:"-"`
	Action      string `json:"-"`
}
