package http

import (
	"activity/services/notification-service/internal/notification/domain"
	"encoding/json"
)

// EncodeFrame is called once per incoming broadcast, not once per connection.
func EncodeFrame(m domain.Message) (string, error) {
	b, err := json.Marshal(map[string]any{"message_id": m.ID, "display_name": m.Name, "item_name": m.ItemName, "quantity": m.Quantity, "expires_at": m.Expires})
	if err != nil {
		return "", err
	}
	return "event: barrage\nid: " + m.ID + "\ndata: " + string(b) + "\n\n", nil
}
