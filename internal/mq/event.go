package mq

import "encoding/json"

// 通用 Event（适配 Router）
type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// 用来把 payload 序列化
func NewEvent(eventType string, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	return Event{
		Type: eventType,
		Data: data,
	}, nil
}
