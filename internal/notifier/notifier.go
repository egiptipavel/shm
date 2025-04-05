package notifier

import (
	"shm/internal/model"
)

type Notifier interface {
	Notify(result model.CheckResult) error
}
