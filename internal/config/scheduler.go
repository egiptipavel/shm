package config

import "time"

type SchedulerConfig struct {
	IntervalMin time.Duration
	CommonConfig
}

func NewSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		IntervalMin:  time.Duration(getEnvAsInt("SCHEDULER_INTERVAL_MIN", 1)) * time.Minute,
		CommonConfig: NewCommonConfig(),
	}
}
