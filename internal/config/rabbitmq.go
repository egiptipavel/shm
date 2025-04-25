package config

type RabbitMQConfig struct {
	User string
	Pass string
	Host string
	Port string
}

func NewRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		User: getEnv("RABBITMQ_DEFAULT_USER", "guest"),
		Pass: getEnv("RABBITMQ_DEFAULT_PASS", "guest"),
		Host: getEnv("RABBITMQ_NODE_IP_ADDRESS", "rabbitmq"),
		Port: getEnv("RABBITMQ_NODE_PORT", "5672"),
	}
}
