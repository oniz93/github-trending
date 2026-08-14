package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration.
type Config struct {
	RabbitMQURL              string
	RabbitMQUser             string
	RabbitMQPassword         string
	GitHubToken              string
	PostgresHost             string
	PostgresPort             string
	PostgresUser             string
	PostgresPassword         string
	PostgresDB               string
	MinioEndpoint            string
	MinioRootUser            string
	MinioRootPassword        string
	ClickHouseHost           string
	ClickHousePort           string
	ClickHouseUser           string
	ClickHousePassword       string
	ClickHouseDB             string
	RedisHost                string
	RedisPort                string
	RedisPassword            string
	QdrantHost               string
	QdrantPort               int
	LastUpdateCut            time.Duration
	SimilarityListSize       int
	EmbeddingApiMaxInstances int
	EmbeddingApiIdleTimeout  int
	Debug                    bool
	TwitterApiKey            string
	TwitterApiSecretKey      string
	TwitterAccessToken       string
	TwitterAccessSecret      string
}

// ParseDuration parses a duration string with support for "months".
func ParseDuration(durationStr string) (time.Duration, error) {
	if strings.HasSuffix(durationStr, " months") {
		monthsStr := strings.TrimSuffix(durationStr, " months")
		months, err := strconv.Atoi(monthsStr)
		if err != nil {
			return 0, fmt.Errorf("invalid number of months: %s", monthsStr)
		}
		return time.Duration(months) * 30 * 24 * time.Hour, nil
	}
	if strings.HasSuffix(durationStr, " years") {
		yearsStr := strings.TrimSuffix(durationStr, " years")
		years, err := strconv.Atoi(yearsStr)
		if err != nil {
			return 0, fmt.Errorf("invalid number of years: %s", yearsStr)
		}
		return time.Duration(years) * 365 * 24 * time.Hour, nil
	}
	return time.ParseDuration(durationStr)
}

// getEnvInt reads an integer environment variable, falling back to a default
// when it is unset. It only returns an error when the variable is set to a
// non-integer value.
func getEnvInt(name string, fallback int) (int, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

// getEnvDuration reads a duration environment variable, falling back to a
// default when it is unset. It only returns an error when the variable is set
// to an invalid duration.
func getEnvDuration(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	return ParseDuration(raw)
}

// Load loads configuration from the environment. Optional settings fall back to
// sensible defaults so services that do not use every subsystem can still start.
func Load() (*Config, error) {
	lastUpdateCut, err := getEnvDuration("LAST_UPDATE_CUT", 24*30*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid LAST_UPDATE_CUT duration: %w", err)
	}

	similarityListSize, err := getEnvInt("SIMILARITY_LIST_SIZE", 200)
	if err != nil {
		return nil, fmt.Errorf("invalid SIMILARITY_LIST_SIZE: %w", err)
	}

	qdrantPort, err := getEnvInt("QDRANT_PORT", 6333)
	if err != nil {
		return nil, fmt.Errorf("invalid QDRANT_PORT: %w", err)
	}

	embeddingApiMaxInstances, err := getEnvInt("EMBEDDING_API_MAX_INSTANCES", 3)
	if err != nil {
		return nil, fmt.Errorf("invalid EMBEDDING_API_MAX_INSTANCES: %w", err)
	}

	embeddingApiIdleTimeout, err := getEnvInt("EMBEDDING_API_IDLE_TIMEOUT", 600)
	if err != nil {
		return nil, fmt.Errorf("invalid EMBEDDING_API_IDLE_TIMEOUT: %w", err)
	}

	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))

	config := &Config{
		RabbitMQURL:              os.Getenv("RABBITMQ_URL"),
		RabbitMQUser:             os.Getenv("RABBITMQ_DEFAULT_USER"),
		RabbitMQPassword:         os.Getenv("RABBITMQ_DEFAULT_PASS"),
		GitHubToken:              os.Getenv("GITHUB_TOKEN"),
		PostgresHost:             os.Getenv("POSTGRES_HOST"),
		PostgresPort:             postgresPort,
		PostgresUser:             os.Getenv("POSTGRES_USER"),
		PostgresPassword:         os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:               os.Getenv("POSTGRES_DB"),
		MinioEndpoint:            os.Getenv("MINIO_ENDPOINT"),
		MinioRootUser:            os.Getenv("MINIO_ROOT_USER"),
		MinioRootPassword:        os.Getenv("MINIO_ROOT_PASSWORD"),
		ClickHouseHost:           os.Getenv("CLICKHOUSE_HOST"),
		ClickHousePort:           os.Getenv("CLICKHOUSE_PORT"),
		ClickHouseUser:           os.Getenv("CLICKHOUSE_USER"),
		ClickHousePassword:       os.Getenv("CLICKHOUSE_PASSWORD"),
		ClickHouseDB:             os.Getenv("CLICKHOUSE_DB"),
		RedisHost:                os.Getenv("REDIS_HOST"),
		RedisPort:                os.Getenv("REDIS_PORT"),
		RedisPassword:            os.Getenv("REDIS_PASSWORD"),
		QdrantHost:               os.Getenv("QDRANT_HOST"),
		QdrantPort:               qdrantPort,
		LastUpdateCut:            lastUpdateCut,
		SimilarityListSize:       similarityListSize,
		EmbeddingApiMaxInstances: embeddingApiMaxInstances,
		EmbeddingApiIdleTimeout:  embeddingApiIdleTimeout,
		Debug:                    debug,
		TwitterApiKey:            os.Getenv("TWITTER_API_KEY"),
		TwitterApiSecretKey:      os.Getenv("TWITTER_API_SECRET_KEY"),
		TwitterAccessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
		TwitterAccessSecret:      os.Getenv("TWITTER_ACCESS_SECRET"),
	}

	if os.Getenv("LOCAL_ENV") == "true" {
		config.RabbitMQURL = "amqp://" + config.RabbitMQUser + ":" + config.RabbitMQPassword + "@localhost:5672/"
	}
	return config, nil
}
