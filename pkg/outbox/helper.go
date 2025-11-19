package outbox

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// InsertEventInTx 在事务中插入事件到 outbox（辅助函数）
func InsertEventInTx(
	ctx context.Context,
	tx pgx.Tx,
	repo *Repository,
	aggregateType string,
	aggregateID *int64,
	routingKey string,
	payload interface{},
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := &Event{
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		RoutingKey:    routingKey,
		Payload:       payloadJSON,
		Status:        "pending",
	}

	return repo.InsertEvent(ctx, tx, event)
}
