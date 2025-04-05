package model

import (
	"database/sql"
	"time"
)

type CheckResult struct {
	Site    Site          `json:"site"`
	Time    time.Time     `json:"time"`
	Latency sql.NullInt64 `json:"latency"`
	Code    sql.NullInt64 `json:"code"`
}

func (c *CheckResult) IsSuccessful() bool {
	return c.Code.Valid && c.Code.Int64 == 200
}
