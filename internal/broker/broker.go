package broker

import (
	"context"
	"shm/internal/model"
)

type MessageBroker interface {
	ConsumeSites(ctx context.Context) (<-chan model.Site, error)
	ConsumeResults(ctx context.Context) (<-chan model.CheckResult, error)
	ConsumeNotifications(ctx context.Context) (<-chan model.Notification, error)

	PublishSite(ctx context.Context, site model.Site) error
	PublishResult(ctx context.Context, result model.CheckResult) error
	PublishNotification(ctx context.Context, notification model.Notification) error

	Close()
}
