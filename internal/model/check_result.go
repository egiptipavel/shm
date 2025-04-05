package model

import (
	"database/sql"
	"time"
)

type CheckResult struct {
	Site    Site
	Time    time.Time
	Latency sql.NullInt64
	Code    sql.NullInt64
}

func (c *CheckResult) IsSuccessful() bool {
	return c.Code.Valid && c.Code.Int64 == 200
}
