package monitor

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"shm/internal/config"
	"shm/internal/notifier"
	"shm/internal/storage"
	"sync"
	"syscall"
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
	var wg sync.WaitGroup
	sites := make(chan storage.Site)
	done := make(chan struct{})
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()

		loop1:
			for {
				select {
				case <-done:
					break loop1
				default:
					site, ok := <-sites
					if !ok {
						break loop1
					}
					m.monitorSite(site)
				}
			}
		}()
	}

loop2:
	for {
		start := time.Now()

		s, err := m.getSites()
		if err != nil {
			slog.Error("failed to get sites", slog.String("error", err.Error()))
			break
		}

		for _, site := range s {
			select {
			case s := <-exit:
				slog.Info("exit signal was received", slog.String("signal", s.String()))
				break loop2
			default:
				sites <- site
			}
		}

		select {
		case <-time.After(time.Duration(m.config.IntervalMins)*time.Minute - time.Since(start)):
		case s := <-exit:
			slog.Info("exit signal was received", slog.String("signal", s.String()))
			break loop2
		}
	}

	close(done)
	close(sites)

	wg.Wait()
}

func (m *Monitor) getSites() ([]storage.Site, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.storage.GetAllMonitoredSites(ctx)
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
