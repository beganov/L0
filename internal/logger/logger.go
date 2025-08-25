package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(level string) {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"}

	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	l, err := zerolog.ParseLevel(level)
	if err != nil {
		l = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(l)
}

func Info(msg string, fields ...interface{}) {
	log.Info().Fields(fields).Msg(msg)
}

func Error(err error, msg string) {
	log.Error().Err(err).Msg(msg)
}

func Fatal(err error, msg string) {
	log.Fatal().Err(err).Msg(msg)
}

func Debug(msg string, fields ...interface{}) {
	log.Debug().Fields(fields).Msg(msg)
}
