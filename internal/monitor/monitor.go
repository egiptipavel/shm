package monitor

import (
	"context"
	"log"
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
	var wg sync.WaitGroup

	for _, site := range m.config.Sites {
		wg.Add(1)
		go func(site string) {
			defer wg.Done()
			m.monitorSite(site)
		}(site)
	}

	wg.Wait()
}

func (m *Monitor) monitorSite(url string) {
	for {
		result, err := m.checkSite(url)
		if err != nil {
			log.Printf("failed to check %s site: %s", url, err)
		} else {
			log.Printf("successful checking of %s site: code %d, latency %dms", url, result.Code, result.Latency)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = m.storage.AddResult(ctx, result)
		cancel()
		if err != nil {
			log.Printf("failed to add check result to storage: %s", err)
			return
		}

		m.notifier.Notify(result)

		time.Sleep(time.Duration(m.config.Interval) * time.Second)
	}
}

func (m *Monitor) checkSite(url string) (result storage.CheckResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return storage.CheckResult{Url: url, Time: start, Latency: 0, Code: 0}, err
	}

	defer resp.Body.Close()

	return storage.CheckResult{Url: url, Time: start, Latency: latency, Code: resp.StatusCode}, nil
}
