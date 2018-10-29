package server

import (
	"github.com/AleksandrKuts/youtumemeter-service/collector/config"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func init() {
	log = config.Logger
}
