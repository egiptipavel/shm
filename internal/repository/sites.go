package repository

import (
	"context"
	"shm/internal/model"
)

type SitesProvider interface {
	AddSite(ctx context.Context, url string) error
	AddSiteFromChat(ctx context.Context, chatId int64, url string) error

	DeleteSiteById(ctx context.Context, siteId int64) error
	DeleteSiteFromChat(ctx context.Context, chatId int64, url string) error

	GetSiteById(ctx context.Context, siteId int64) (model.Site, error)
	GetAllSites(ctx context.Context) ([]model.Site, error)
	GetAllMonitoredSites(ctx context.Context) ([]model.Site, error)
	GetAllSitesByChatId(ctx context.Context, chatId int64) ([]model.Site, error)
}
