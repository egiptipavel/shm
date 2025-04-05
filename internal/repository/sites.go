package repository

import (
	"context"
	"database/sql"
	"errors"
	"shm/internal/model"
)

type Sites struct {
	db *sql.DB
}

func NewSitesRepo(db *sql.DB) *Sites {
	return &Sites{db}
}

func (s *Sites) AddSite(ctx context.Context, url string) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO sites (url) VALUES (?) ON CONFLICT DO NOTHING",
		url,
	)
	return err
}

func (s *Sites) AddSiteFromChat(ctx context.Context, chatId int64, url string) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO sites (url) VALUES (?) ON CONFLICT DO NOTHING",
		url,
	)
	if err != nil {
		return err
	}

	var siteId int64
	err = s.db.QueryRowContext(ctx, "SELECT id FROM sites WHERE url = ?", url).Scan(&siteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		"INSERT INTO chat_to_site (chat_id, site_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		chatId, siteId,
	)
	return err
}

func (s *Sites) DeleteSiteById(ctx context.Context, siteId int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sites WHERE id = ?", siteId)
	return err
}

func (s *Sites) DeleteSiteFromChat(ctx context.Context, chatId int64, url string) error {
	var siteId int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM sites WHERE url = ?", url).Scan(&siteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		"DELETE FROM chat_to_site WHERE chat_id = ? AND site_id = ?",
		chatId, siteId,
	)
	return err
}

func (s *Sites) GetSiteById(ctx context.Context, siteId int64) (model.Site, error) {
	var site model.Site
	row := s.db.QueryRowContext(ctx, "SELECT * FROM sites WHERE id = ?", siteId)
	err := row.Scan(&site.Id, &site.Url)
	return site, err
}

func (s *Sites) GetAllSites(ctx context.Context) ([]model.Site, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * FROM sites")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []model.Site
	for rows.Next() {
		var site model.Site

		err = rows.Scan(&site.Id, &site.Url)
		if err != nil {
			return nil, err
		}

		sites = append(sites, site)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func (s *Sites) GetAllMonitoredSites(ctx context.Context) ([]model.Site, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT DISTINCT s.id, s.url FROM sites AS s JOIN chat_to_site AS c ON s.id = c.site_id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []model.Site
	for rows.Next() {
		var site model.Site

		err = rows.Scan(&site.Id, &site.Url)
		if err != nil {
			return nil, err
		}

		sites = append(sites, site)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func (s *Sites) GetAllSitesByChatId(ctx context.Context, chatId int64) ([]model.Site, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT s.id, s.url 
		FROM chat_to_site as c
		JOIN sites as s
		ON c.site_id = s.id
		WHERE c.chat_id = ?`,
		chatId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []model.Site
	for rows.Next() {
		var site model.Site

		err = rows.Scan(&site.Id, &site.Url)
		if err != nil {
			return nil, err
		}

		sites = append(sites, site)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}
