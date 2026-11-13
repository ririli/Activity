package domain

type Prize struct {
	Code     string  `json:"code"`
	ID       uint64  `json:"item_id,string"`
	Name     string  `json:"name"`
	Quantity int64   `json:"quantity"`
	Weight   int64   `json:"weight"`
	Choices  []Prize `json:"choices,omitempty"`
}
