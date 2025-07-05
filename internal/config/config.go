package config

import "os"

// Config holds application configuration.
type Config struct {
	RabbitMQURL        string
	RabbitMQUser       string
	RabbitMQPassword   string
	GitHubToken        string
	PostgresHost       string
	PostgresUser       string
	PostgresPassword   string
	PostgresDB         string
	MinioEndpoint      string
	MinioRootUser      string
	MinioRootPassword  string
	ClickHouseHost     string
	ClickHousePort     string
	ClickHouseUser     string
	ClickHousePassword string
	ClickHouseDB       string
	RedisHost          string
	RedisPort          string
	RedisPassword      string
	MilvusHost         string
	MilvusPort         string
	LastUpdateCut      string
	SimilarityListSize string
}

// Load loads configuration from the environment.
func Load() (*Config, error) {
	config := &Config{
		RabbitMQURL:        os.Getenv("RABBITMQ_URL"),
		RabbitMQUser:       os.Getenv("RABBITMQ_DEFAULT_USER"),
		RabbitMQPassword:   os.Getenv("RABBITMQ_DEFAULT_PASS"),
		GitHubToken:        os.Getenv("GITHUB_TOKEN"),
		PostgresHost:       os.Getenv("POSTGRES_HOST"),
		PostgresUser:       os.Getenv("POSTGRES_USER"),
		PostgresPassword:   os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:         os.Getenv("POSTGRES_DB"),
		MinioEndpoint:      os.Getenv("MINIO_ENDPOINT"),
		MinioRootUser:      os.Getenv("MINIO_ROOT_USER"),
		MinioRootPassword:  os.Getenv("MINIO_ROOT_PASSWORD"),
		ClickHouseHost:     os.Getenv("CLICKHOUSE_HOST"),
		ClickHousePort:     os.Getenv("CLICKHOUSE_PORT"),
		ClickHouseUser:     os.Getenv("CLICKHOUSE_USER"),
		ClickHousePassword: os.Getenv("CLICKHOUSE_PASSWORD"),
		ClickHouseDB:       os.Getenv("CLICKHOUSE_DB"),
		RedisHost:          os.Getenv("REDIS_HOST"),
		RedisPort:          os.Getenv("REDIS_PORT"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		MilvusHost:         os.Getenv("MILVUS_HOST"),
		MilvusPort:         os.Getenv("MILVUS_PORT"),
		LastUpdateCut:      os.Getenv("LAST_UPDATE_CUT"),
		SimilarityListSize: os.Getenv("SIMILARITY_LIST_SIZE"),
	}

	if os.Getenv("LOCAL_ENV") == "true" {
		config.RabbitMQURL = "amqp://" + config.RabbitMQUser + ":" + config.RabbitMQPassword + "@localhost:5672/"
	}
	return config, nil
}
