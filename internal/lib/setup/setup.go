package setup

import (
	"fmt"
	"log/slog"
	"os"
	"shm/internal/broker"
	"shm/internal/config"
	"shm/internal/db"
	"shm/internal/lib/sl"
)

type DatabaseCreator = func() db.Database
type BrokerCreator = func() broker.MessageBroker

var drivers = map[string]DatabaseCreator{
	"postgres": func() db.Database {
		return connectToPostgres(config.NewPostgresConfig())
	},
	"sqlite": func() db.Database {
		return connectToSQLite(config.NewSQLiteConfig())
	},
}

var brokers = map[string]BrokerCreator{
	"rabbitmq": func() broker.MessageBroker {
		return connectToRabbitMQ(config.NewRabbitMQConfig())
	},
}

func ConnectToDatabase(driverName string) db.Database {
	dbCreator, exists := drivers[driverName]
	if !exists {
		slog.Error("unknown database driver", slog.String("driver", driverName))
		os.Exit(1)
	}
	return dbCreator()
}

func connectToSQLite(config config.SQLiteConfig) *db.SQLite {
	slog.Info("connecting to SQLite")
	db, err := db.NewSQLite(config.File)
	if err != nil {
		slog.Error("failed to create database", sl.Error(err))
		os.Exit(1)
	}
	return db
}

func connectToPostgres(config config.PostgresConfig) *db.Postgres {
	slog.Info("connecting to PostgreSQL")
	url := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		config.User, config.Pass, config.Host, config.Port, config.Db,
	)
	db, err := db.NewPostgres(url)
	if err != nil {
		slog.Error("failed to create database", sl.Error(err))
		os.Exit(1)
	}
	return db
}

func ConnectToMessageBroker(brokerName string) broker.MessageBroker {
	brokerCreator, exists := brokers[brokerName]
	if !exists {
		slog.Error("unknown message broker", slog.String("message_broker", brokerName))
		os.Exit(1)
	}
	return brokerCreator()
}

func connectToRabbitMQ(config config.RabbitMQConfig) *broker.RabbitMQ {
	slog.Info("connecting to RabbitMQ")
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", config.User, config.Pass, config.Host, config.Port)
	broker, err := broker.NewRabbitMQ(url)
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", sl.Error(err))
		os.Exit(1)
	}
	return broker
}
