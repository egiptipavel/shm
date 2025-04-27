package repository

import (
	"context"
	"database/sql"
	"shm/internal/model"
)

type Results struct {
	db *sql.DB
}

func NewResultsRepo(db *sql.DB) *Results {
	return &Results{db}
}

func (r *Results) AddResult(ctx context.Context, result model.CheckResult) error {
	_, err := r.db.ExecContext(
		ctx,
		"INSERT INTO check_results (site_id, time, latency, code) VALUES (?, ?, ?, ?)",
		result.Site.Id, result.Time, result.Latency, result.Code,
	)

	return err
}

func (r *Results) GetNLastResultsForSite(
	ctx context.Context,
	site model.Site,
	n int,
) ([]model.CheckResult, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT s.id, s.url, c.time, c.latency, c.code
		FROM check_results AS c
		JOIN sites AS s
		ON c.site_id = s.id
		WHERE s.id = ?
		ORDER BY c.time DESC
		LIMIT ?`,
		site.Id, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.CheckResult
	for rows.Next() {
		var result model.CheckResult

		err = rows.Scan(
			&result.Site.Id,
			&result.Site.Url,
			&result.Time,
			&result.Latency,
			&result.Code,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func (r *Results) GetSecondToLastSuccessfulResultForSite(
	ctx context.Context,
	site model.Site,
) (model.CheckResult, error) {
	var result model.CheckResult
	err := r.db.QueryRowContext(ctx,
		`SELECT s.id, s.url, c.time, c.latency, c.code
		FROM check_results as c
		JOIN sites as s
		ON c.site_id = s.id
		WHERE s.id = ? AND c.code = 200
		ORDER BY c.time DESC
		LIMIT 1 OFFSET 1`,
		site.Id,
	).Scan(
		&result.Site.Id,
		&result.Site.Url,
		&result.Time,
		&result.Latency,
		&result.Code,
	)
	return result, err
}
