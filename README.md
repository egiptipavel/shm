# Site Health Monitor

Site Health Monitor is an automated monitoring tool designed to track the health and status of websites. It provides real-time monitoring and alerts, ensuring that your sites remains available and performing optimally.

## Technologies Used

* Go 1.23.5
* RabbitMQ as message broker
* PostgreSQL as database
* Telegram for notification channel
* Goose migration tool

## Getting Started

To get started with Site Health Monitor, follow these steps:

1. Clone the repository: `git clone https://github.com/egiptipavel/shm.git`
2. Create file `rabbitmq-env.conf` in root of project with content:
```
RABBITMQ_DEFAULT_USER=rabbitmq
RABBITMQ_DEFAULT_PASS=rabbitmq
```
3. Create file `.env` in root of project with content:
```
TELEGRAM_TOKEN=your_telegram_bot_token
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=postgres_db
```
4. Build and run the containers: `docker compose up -d` (or `docker compose up` to run in foreground)
