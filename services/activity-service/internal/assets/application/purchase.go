package application

import (
	"activity/pkg/platform"
	"activity/services/activity-service/internal/appstate"
	"activity/services/activity-service/internal/uow"
	"errors"
	"fmt"
	"math"
)

// Purchase owns the product catalog decisions and wallet changes for a
// top-level purchase. Quota/order persistence is delegated to appstate.
func Purchase(s *appstate.State, c appstate.Command) (any, error) {
	oldID, oldType, oldProduct, oldPetals, err := s.FindPurchase(c.Operation)
	if err == nil {
		if oldProduct != c.Product || (oldType == "TOP_UP" && oldPetals != c.Petals) {
			return nil, platform.Fail(409, "PURCHASE_CONFLICT")
		}
		return map[string]string{"order_id": fmt.Sprint(oldID)}, nil
	}
	if !errors.Is(err, uow.ErrNoRows) {
		return nil, err
	}
	var cost, petals int64
	limited := true
	kind := "SHOP"
	switch c.Product {
	case "5":
		cost = 300
		petals = 5
		limited = false
	case "52":
		cost = 3000
		petals = 52
	case "525":
		cost = 30000
		petals = 525
	case "1580":
		cost = 90000
		petals = 1580
	case "gift":
		kind = "SIMULATED_GIFT"
		petals = 100
	case "top-up":
		kind = "TOP_UP"
		limited = false
		petals = c.Petals
		if petals <= 0 || petals > math.MaxInt64/60 {
			return nil, platform.Fail(400, "INVALID_PETALS")
		}
		cost = petals * 60
	default:
		return nil, platform.Fail(400, "INVALID_PRODUCT")
	}
	if limited {
		n, e := s.CountPurchaseQuota(c.Product)
		if e != nil {
			return nil, e
		}
		if n > 0 {
			return nil, platform.Fail(409, "ALREADY_PURCHASED")
		}
	}
	if e := s.Asset("GOLD", -cost, "PURCHASE_COST", "PURCHASE"); e != nil {
		return nil, e
	}
	if e := s.Asset("PETAL", petals, "PURCHASE_GRANT", "PURCHASE"); e != nil {
		return nil, e
	}
	if e := s.InsertPurchase(c.Operation, kind, c.Product, cost, petals, s.Version); e != nil {
		return nil, e
	}
	if limited {
		if e := s.InsertPurchaseQuota(c.Product); e != nil {
			return nil, e
		}
	}
	return map[string]any{"order_id": fmt.Sprint(s.Biz), "gold": s.Gold, "petals": s.Petals}, nil
}
