package config

import (
	"os"
	"strconv"
	"time"

	"github.com/beganov/L0/internal/logger"
)

var (
	KafkaBroker  string
	KafkaTopic   string
	KafkaGroupID string

	PostgresURL string

	CacheCap int

	HttpAddr string

	HttpTimeOut   time.Duration
	SelectTimeOut time.Duration
	InsertTimeOut time.Duration
	KafkaTimeOut  time.Duration

	MigrationPath string
)

func VarsInit() {

	KafkaBroker = os.Getenv("KAFKA_BROKER")
	KafkaTopic = os.Getenv("KAFKA_TOPIC")
	KafkaGroupID = os.Getenv("KAFKA_GROUP_ID")
	PostgresURL = os.Getenv("POSTGRES_URL")
	HttpAddr = os.Getenv("HTTP_ADDR")

	var err error
	CacheCap, err = strconv.Atoi(os.Getenv("CACHE_CAP"))
	if err != nil {
		logger.Fatal(err, "CACHE_CAP is not number")
	}

	httpTimeoutSec, err := strconv.Atoi(os.Getenv("HTTP_TIMEOUT"))
	if err != nil {
		logger.Fatal(err, "HTTP_TIMEOUT is not number")
	}
	selectTimeoutSec, err := strconv.Atoi(os.Getenv("SELECT_TIMEOUT"))
	if err != nil {
		logger.Fatal(err, "SELECT_TIMEOUT is not number")
	}
	insertTimeoutSec, err := strconv.Atoi(os.Getenv("INSERT_TIMEOUT"))
	if err != nil {
		logger.Fatal(err, "INSERT_TIMEOUT is not number")
	}
	kafkaTimeoutSec, err := strconv.Atoi(os.Getenv("KAFKA_TIMEOUT"))
	if err != nil {
		logger.Fatal(err, "KAFKA_TIMEOUT is not number")
	}

	HttpTimeOut = time.Duration(httpTimeoutSec) * time.Second
	SelectTimeOut = time.Duration(selectTimeoutSec) * time.Second
	InsertTimeOut = time.Duration(insertTimeoutSec) * time.Second
	KafkaTimeOut = time.Duration(kafkaTimeoutSec) * time.Second
	MigrationPath = os.Getenv("MIGRATION_PATH")
}
