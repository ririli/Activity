package application

import (
	assetsdomain "activity/services/activity-service/internal/assets/domain"
	"activity/services/activity-service/internal/fault"
	"errors"
	"fmt"
)

// purchase coordinates product eligibility, classified wallet changes and order persistence.
func purchase(s *work, c PurchaseCommand) (any, error) {
	oldID, oldEdition, oldType, oldProduct, oldPetals, err := s.session.Assets.FindPurchase(c.Operation)
	if err == nil {
		if oldEdition != s.scope.Edition || oldProduct != c.Product || (oldType == "TOP_UP" && oldPetals != c.Petals) {
			return nil, fault.New("PURCHASE_CONFLICT")
		}
		return map[string]string{"order_id": fmt.Sprint(oldID)}, nil
	}
	if !errors.Is(err, fault.ErrNotFound) {
		return nil, err
	}
	product, err := assetsdomain.Quote(c.Product, c.Petals)
	if err != nil {
		return nil, err
	}
	cost, petals, limited, kind := product.Cost, product.Petals, product.Limited, product.Kind
	if limited {
		n, e := s.session.Assets.CountPurchaseQuota(c.Product)
		if e != nil {
			return nil, e
		}
		if n > 0 {
			return nil, fault.New("ALREADY_PURCHASED")
		}
	}
	if e := s.Asset("GOLD", -cost, "PURCHASE_COST", "PURCHASE"); e != nil {
		return nil, e
	}
	if e := s.Asset("PETAL", petals, "PURCHASE_GRANT", "PURCHASE"); e != nil {
		return nil, e
	}
	if e := s.session.Assets.InsertPurchase(c.Operation, kind, c.Product, cost, petals, s.progress.Version); e != nil {
		return nil, e
	}
	if limited {
		if e := s.session.Assets.InsertPurchaseQuota(c.Product); e != nil {
			return nil, e
		}
	}
	return map[string]any{"order_id": fmt.Sprint(s.scope.Biz), "gold": s.wallet.Gold, "petals": s.wallet.Petals}, nil
}
