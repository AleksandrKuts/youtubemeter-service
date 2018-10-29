package server

import (
	"github.com/AleksandrKuts/go/youtubemeter/metercollect/config"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func init() {
	log = config.Logger
}
