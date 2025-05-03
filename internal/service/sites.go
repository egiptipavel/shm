package service

import (
	"context"
	"database/sql"
	"errors"
	"shm/internal/config"
	"shm/internal/model"
	"shm/internal/repository"
)

type SitesService struct {
	sites  repository.SitesProvider
	config config.CommonConfig
}

func NewSitesService(sites repository.SitesProvider, config config.CommonConfig) *SitesService {
	return &SitesService{
		sites:  sites,
		config: config,
	}
}

func (s *SitesService) AddSite(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	return s.sites.AddSite(ctx, url)
}

func (s *SitesService) AddSiteFromChat(ctx context.Context, chatId int64, url string) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	return s.sites.AddSiteFromChat(ctx, chatId, url)
}

func (s *SitesService) DeleteSiteById(ctx context.Context, siteId int64) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	return s.sites.DeleteSiteById(ctx, siteId)
}

func (s *SitesService) DeleteSiteFromChat(ctx context.Context, chatId int64, url string) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	return s.sites.DeleteSiteFromChat(ctx, chatId, url)
}

func (s *SitesService) GetSiteById(ctx context.Context, siteId int64) (*model.Site, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	site, err := s.sites.GetSiteById(ctx, siteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &site, nil
}

func (s *SitesService) GetAllSites(ctx context.Context) ([]model.Site, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	return s.sites.GetAllSites(ctx)
}

func (s *SitesService) GetAllMonitoredSites(ctx context.Context) ([]model.Site, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	return s.sites.GetAllMonitoredSites(ctx)
}

func (s *SitesService) GetAllSitesByChatId(ctx context.Context, chatId int64) ([]model.Site, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.DbQueryTimeoutSec)
	defer cancel()

	return s.sites.GetAllSitesByChatId(ctx, chatId)
}
