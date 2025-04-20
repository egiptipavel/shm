package sl

import (
	"log/slog"
	"shm/internal/model"
)

func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func CheckResult(result model.CheckResult) slog.Attr {
	return slog.Group("check_result",
		slog.String("url", result.Site.Url),
		slog.Int64("code", result.Code.Int64),
		slog.Int64("latency_ms", result.Latency.Int64),
	)
}

func Site(site model.Site) slog.Attr {
	return slog.Group("site",
		slog.Int64("id", site.Id),
		slog.String("url", site.Url),
	)
}

func Notification(notification model.Notification) slog.Attr {
	return slog.Group("notification",
		slog.String("url", notification.Url),
		slog.String("message", notification.Message),
	)
}
