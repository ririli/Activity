package effects

import "activity/pkg/platform"

type Effects struct {
	Notify chan []byte
}

func (e Effects) Enqueue(messages []map[string]any) {
	if e.Notify == nil {
		return
	}
	for _, message := range messages {
		select {
		case e.Notify <- platform.JSON(message):
		default:
		}
	}
}
