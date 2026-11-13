package domain

import (
	"activity/services/activity-service/internal/fault"
	"math"
)

type Product struct {
	Kind         string
	Cost, Petals int64
	Limited      bool
}

func Quote(code string, petals int64) (Product, error) {
	p := Product{Kind: "SHOP", Limited: true}
	switch code {
	case "5":
		p.Cost = 300
		p.Petals = 5
		p.Limited = false
	case "52":
		p.Cost = 3000
		p.Petals = 52
	case "525":
		p.Cost = 30000
		p.Petals = 525
	case "1580":
		p.Cost = 90000
		p.Petals = 1580
	case "gift":
		p.Kind = "SIMULATED_GIFT"
		p.Petals = 100
	case "top-up":
		if petals <= 0 || petals > math.MaxInt64/60 {
			return p, fault.New("INVALID_PETALS")
		}
		p.Kind = "TOP_UP"
		p.Limited = false
		p.Petals = petals
		p.Cost = petals * 60
	default:
		return p, fault.New("INVALID_PRODUCT")
	}
	return p, nil
}
