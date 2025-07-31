package auth

import (
	"github.com/charmbracelet/log"
	"github.com/knadh/koanf/v2"
	"github.com/mc-botnet/mc-botnet-server/internal/database"
	"github.com/mc-botnet/mc-botnet-server/internal/logger"
)

type Service struct {
	l     *log.Logger
	conf  *koanf.Koanf
	store database.Store
}

func NewService(conf *koanf.Koanf, store database.Store) *Service {
	return &Service{logger.NewLogger("auth", log.InfoLevel), conf, store}
}
