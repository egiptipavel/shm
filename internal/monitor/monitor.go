package monitor

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"shm/internal/config"
	"shm/internal/notifier"
	"shm/internal/storage"
	"sync"
	"time"
)

type Monitor struct {
	storage  *storage.Storage
	notifier notifier.Notifier
	config   config.Config
}

func New(storage *storage.Storage, notifier notifier.Notifier, config config.Config) *Monitor {
	return &Monitor{storage, notifier, config}
}

func (m *Monitor) Start() {
	m.startMonitoring()
}

func (m *Monitor) startMonitoring() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		sites, err := m.storage.GetAllMonitoredSites(ctx)
		cancel()
		if err != nil {
			slog.Error("failed to get all sites", slog.String("error", err.Error()))
			break
		}

		var wg sync.WaitGroup
		for _, site := range sites {
			wg.Add(1)
			go func(site storage.Site) {
				defer wg.Done()
				m.monitorSite(site)
			}(site)
		}
		wg.Wait()

		time.Sleep(time.Duration(m.config.IntervalMins) * time.Minute)
	}
}

func (m *Monitor) monitorSite(site storage.Site) {
	result, err := m.checkSite(site)
	if err != nil {
		slog.Error(
			"unsuccessful checking of site",
			slog.String("url", site.Url),
			slog.String("error", err.Error()),
		)
	} else {
		slog.Info(
			"successful checking of site",
			slog.String("url", site.Url),
			slog.Int64("code", result.Code.Int64),
			slog.Int64("latency_ms", result.Latency.Int64),
		)
	}

	if m.notifier != nil {
		if err = m.notifier.Notify(result); err != nil {
			slog.Error("failed to notify", slog.String("error", err.Error()))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = m.storage.AddResult(ctx, result)
	cancel()
	if err != nil {
		slog.Error("failed to add check result to storage", slog.String("error", err.Error()))
	}
}

func (m *Monitor) checkSite(site storage.Site) (result storage.CheckResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "GET", site.Url, nil)
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return storage.CheckResult{
			Site:    site,
			Time:    start,
			Latency: sql.NullInt64{},
			Code:    sql.NullInt64{},
		}, err
	}

	defer resp.Body.Close()

	return storage.CheckResult{
		Site: site,
		Time: start,
		Latency: sql.NullInt64{
			Int64: latency,
			Valid: true,
		},
		Code: sql.NullInt64{
			Int64: int64(resp.StatusCode),
			Valid: true,
		},
	}, nil
}
