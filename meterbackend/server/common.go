package server

import (
	"github.com/AleksandrKuts/go/youtubemeter/meterbackend/config"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func init() {
	log = config.Logger
}
