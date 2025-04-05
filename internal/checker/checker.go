package checker

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"shm/internal/model"
	"shm/internal/repository"
	"sync"
	"syscall"
	"time"
)

type Checker struct {
	results      *repository.Results
	sites        *repository.Sites
	intervalMins int
}

func New(db *sql.DB, intervalMins int) *Checker {
	return &Checker{
		repository.NewResultsRepo(db),
		repository.NewSitesRepo(db),
		intervalMins,
	}
}

func (m *Checker) Start() {
	var wg sync.WaitGroup
	sites := make(chan model.Site)
	done := make(chan struct{})
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	for range 1000 {
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
		case <-time.After(time.Duration(m.intervalMins)*time.Minute - time.Since(start)):
		case s := <-exit:
			slog.Info("exit signal was received", slog.String("signal", s.String()))
			break loop2
		}
	}

	close(sites)

	wg.Wait()
}

func (m *Checker) getSites() ([]model.Site, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return m.sites.GetAllMonitoredSites(ctx)
}

func (m *Checker) monitorSite(site model.Site) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = m.results.AddResult(ctx, result)
	cancel()
	if err != nil {
		slog.Error("failed to add check result to storage", slog.String("error", err.Error()))
	}
}

func (m *Checker) checkSite(site model.Site) (result model.CheckResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "GET", site.Url, nil)
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return model.CheckResult{
			Site:    site,
			Time:    start,
			Latency: sql.NullInt64{},
			Code:    sql.NullInt64{},
		}, err
	}

	defer resp.Body.Close()

	return model.CheckResult{
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
