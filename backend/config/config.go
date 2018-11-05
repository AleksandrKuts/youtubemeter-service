package config

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
	"github.com/vharitonsky/iniflags"
	"fmt"
	"strings"	
)

var (
	// Get command options
	Addr = flag.String("Addr", "0.0.0.0:3000", "server's port. Example: 0.0.0.0:3000")
	Timeout = flag.Duration("timeout", time.Second * 15, "the duration for which the server wait for existing connections to finish - e.g. 15s or 1m")
	ListenAdmin = flag.Bool("ListenAdmin", false, "Activate the playlist administration service")
	Origin = flag.String("Origin", "*", "Source from which the request to the service is allowed. Example: 0.0.0.0:4200")
	MaxViewVideosInPlayLists = flag.Int("MaxViewVideosInPlayLists", 30, "The maximum number of videos to display in the playlist. Example: 30")

	PeriodMeterCache = flag.Duration("periodMetricCache", time.Second * 60, "the frequency of checking video meter - e.g. 60s or 1m")
	PeriodCollectionCache = flag.Duration("periodCollectCache", time.Hour * 24 * 14, "the collection period video statistics from the date and time that the video was uploaded- e.g. 336h")

	MaxSizeCacheVideo = flag.Int("MaxSizeCacheVideo", 1000, "The maximum size of videos cache. Example: 100")
	MaxSizeCacheMetrics = flag.Int("MaxSizeCacheMetrics", 1000, "The maximum size of metrics cache. Example: 100")

	debugLevel = flag.String("debugLevel", "info", "debug level: debug, info, warn, error, dpanic, panic, fatal. Example: error")
	Log = flag.String("Log", "backend.log", "log files")
	LogError = flag.String("LogError", "backend_error.log", "log files")

	DBHost = flag.String("dbhost", "localhost", "The database's host to connect to. Values that start with / are for unix")
	DBPort = flag.String("dbport", "5432", "The database's port to bind to")
	DBName = flag.String("dbname", "basename", "The name of the database to connect to")
	DBUser = flag.String("dbuser", "username", "The database's user to sign in as")
	DBPassword = flag.String("dbpasswd", "userpasswd", "The database's user's password")
	DBSSLMode = flag.String("dbsslmode", "disable", "Whether or not to use SSL for the database's host")

	Logger *zap.SugaredLogger	
)


func myTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("02-01-2006 15:04:05"))
}

func init() {

	iniflags.Parse() 

	// set debug level
	var atomicLevel zapcore.Level

	switch *debugLevel {
	case "debug":
		atomicLevel = zapcore.DebugLevel
	case "info":
		atomicLevel = zapcore.InfoLevel
	case "warn":
		atomicLevel = zapcore.WarnLevel
	case "error":
		atomicLevel = zapcore.ErrorLevel
	case "dpanic":
		atomicLevel = zapcore.DPanicLevel
	case "panic":
		atomicLevel = zapcore.PanicLevel
	case "fatal":
		atomicLevel = zapcore.FatalLevel
	default:
		atomicLevel = zapcore.InfoLevel
	}

	// Set loggin systems
	cfg := zap.Config{
		Encoding:         "console",
		Level:            zap.NewAtomicLevelAt(atomicLevel),
		OutputPaths:      strings.Split( *Log, ","),
		ErrorOutputPaths: strings.Split( *Log, ","),
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: myTimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	logger, _ := cfg.Build()
	defer logger.Sync() // flushes buffer, if any

	
	Logger = logger.Sugar()

	Logger.Warnf("debug level=%s", atomicLevel)
	Logger.Warnf("Log=%s", Log)
	Logger.Warnf("LogError=%s", LogError)
	
	Logger.Infof("addr=%s", *Addr)
	Logger.Infof("timeout=%s", *Timeout)
	Logger.Infof("ListenAdmin=%v", *ListenAdmin)
	Logger.Infof("Origin=%v", *Origin)
	Logger.Infof("MaxViewVideosInPlayLists=%v", *MaxViewVideosInPlayLists)
	Logger.Infof("PeriodMeter=%v", *PeriodMeterCache)
	Logger.Infof("PeriodСollection=%v", *PeriodCollectionCache)

	Logger.Infof("MaxSizeCacheVideo=%v", *MaxSizeCacheVideo)
	Logger.Infof("MaxSizeCacheMetrics=%v", *MaxSizeCacheMetrics)

	Logger.Debugf("dbhost=%s", *DBHost)
	Logger.Debugf("dbport=%s", *DBPort)	
	Logger.Debugf("dbname=%s", *DBName)
	Logger.Debugf("dbuser=%s", *DBUser)
	Logger.Debugf("dbpasswd=%s", *DBPassword)
	Logger.Debugf("dbsslmode=%s", *DBSSLMode)
}
