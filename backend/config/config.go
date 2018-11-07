package config

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
	"github.com/vharitonsky/iniflags"
	"strings"	
)

var (
	Addr = flag.String("Addr", "0.0.0.0:3000", "")
	Timeout = flag.Duration("timeout", time.Second * 15, "")
	ListenAdmin = flag.Bool("ListenAdmin", false, "")
	Origin = flag.String("Origin", "*", "")
	MaxViewVideosInPlayLists = flag.Int("MaxViewVideosInPlayLists", 30, "")

	EnableCache = flag.Bool("enableCache", true, "Enable cache?")
	PeriodPlayListCache = flag.Duration("periodPlayListCache", time.Minute * 30, "")
	PeriodMeterCache = flag.Duration("periodMetricCache", time.Second * 60, "")	
	PeriodCollectionCache = flag.Duration("periodCollectCache", time.Hour * 24 * 14, "")
	PeriodVideoCache = flag.Duration("periodVideoCache", time.Minute * 5, "")

	MaxSizeCacheVideo = flag.Int("maxSizeCacheVideo", 1000, "")
	MaxSizeCachePlaylists = flag.Int("maxSizeCachePlaylists", 1000, "")

	debugLevel = flag.String("debugLevel", "info", "")
	Log = flag.String("Log", "backend.log", "")
	LogError = flag.String("LogError", "backend_error.log", "")
	LogTimeFormat = flag.String("LogTimeFormat", "02-01-2006 15:04:05", "")

	DBHost = flag.String("dbhost", "localhost", "")
	DBPort = flag.String("dbport", "5432", "")
	DBName = flag.String("dbname", "basename", "")
	DBUser = flag.String("dbuser", "username", "")
	DBPassword = flag.String("dbpasswd", "userpasswd", "")
	DBSSLMode = flag.String("dbsslmode", "disable", "")

	Logger *zap.SugaredLogger	
)

func myTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(*LogTimeFormat))
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
	Logger.Debugf("Log=%s", Log)
	Logger.Debugf("LogError=%s", LogError)
	Logger.Debugf("LogTimeFormat=%s", LogTimeFormat)
	
	Logger.Debugf("addr=%s", *Addr)
	Logger.Debugf("timeout=%s", *Timeout)
	Logger.Debugf("ListenAdmin=%v", *ListenAdmin)
	Logger.Debugf("Origin=%v", *Origin)
	Logger.Debugf("MaxViewVideosInPlayLists=%v", *MaxViewVideosInPlayLists)
		
	Logger.Debugf("PeriodPlayListCache=%v", *PeriodPlayListCache)
	Logger.Debugf("PeriodVideoCache=%v", *PeriodVideoCache)
	Logger.Debugf("PeriodMeterCache=%v", *PeriodMeterCache)
	Logger.Debugf("PeriodСollectionCache=%v", *PeriodCollectionCache)

	Logger.Debugf("EnableCache=%v", *EnableCache)
	Logger.Debugf("MaxSizeCacheVideo=%v", *MaxSizeCacheVideo)
	Logger.Debugf("MaxSizeCachePlaylists=%v", *MaxSizeCachePlaylists)

	Logger.Debugf("dbhost=%s", *DBHost)
	Logger.Debugf("dbport=%s", *DBPort)	
	Logger.Debugf("dbname=%s", *DBName)
	Logger.Debugf("dbuser=%s", *DBUser)
	Logger.Debugf("dbpasswd=%s", *DBPassword)
	Logger.Debugf("dbsslmode=%s", *DBSSLMode)
}
