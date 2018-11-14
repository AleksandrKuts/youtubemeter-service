package config

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
	"strings"
	"github.com/vharitonsky/iniflags"
)

var (
	// Get command options
	debugLevel = flag.String("debugLevel", "info", "")
	Log = flag.String("Log", "collector.log", "")
	LogError = flag.String("LogError", "collector_error.log", "")

	FileSecret = flag.String("fileToken", "client_secret.json", "")
	CredentialFile = flag.String("fileCredential", "yotubemetric_credential.json", "")
	
	Timeout = flag.Duration("timeout", time.Second * 15, "")

	PeriodPlayList = flag.Duration("periodPlayList", time.Second * 600, "")
	PeriodVideo = flag.Duration("periodVideo", time.Second * 60, "")
	PeriodMeter = flag.Duration("periodMetric", time.Second * 60, "")
	PeriodCount = flag.Duration("periodSaveMetricIdle", time.Hour * 1, "")
	PeriodDeleted = flag.Duration("periodFinalDeletion", time.Hour * 24, "")
	PeriodСollection = flag.Duration("periodCollect", time.Hour * 24 * 14, "")
	MaxRequestCountVideoID = flag.Int("maxRequestCountVideoID", 50, "")
	
	DBHost = flag.String("dbhost", "localhost", "")
	DBPort = flag.String("dbport", "5432", "")
	DBName = flag.String("dbname", "basename", "")
	DBUser = flag.String("dbuser", "username", "")
	DBPassword = flag.String("dbpasswd", "userpasswd", "")
	DBSSLMode = flag.String("dbsslmode", "disable", "")

	Logger *zap.SugaredLogger	
)

func myTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("02-01-2006 15:04:05"))
}

func init() {
	iniflags.Parse() 

	*FileSecret = strings.TrimSpace(*FileSecret)

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
		ErrorOutputPaths: strings.Split( *LogError, ","),
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

	if *MaxRequestCountVideoID < 1 && *MaxRequestCountVideoID > 50 {
		*MaxRequestCountVideoID = 50
	}

	logger, _ := cfg.Build()
	defer logger.Sync() // flushes buffer, if any
	Logger = logger.Sugar()	
	
	Logger.Warnf("debug level=%v", atomicLevel)
	Logger.Infof("Log=%s", Log)
	Logger.Debugf("LogError=%s", LogError)

	Logger.Debugf("fileSecret=%v", *FileSecret)
	Logger.Debugf("timeout=%s", *Timeout)
	
	Logger.Debugf("PeriodPlayList=%v", *PeriodPlayList)
	Logger.Debugf("PeriodVideo=%v", *PeriodVideo)
	Logger.Debugf("PeriodMeter=%v", *PeriodMeter)
	Logger.Debugf("PeriodCount=%v", *PeriodCount)
	Logger.Debugf("PeriodDeleted=%v", *PeriodDeleted)
	Logger.Debugf("PeriodСollection=%v", *PeriodСollection)
	Logger.Debugf("MaxReqestCountVideoID=%v", *MaxRequestCountVideoID)

	Logger.Debugf("dbhost=%s", *DBHost)
	Logger.Debugf("dbport=%s", *DBPort)
	Logger.Debugf("dbname=%s", *DBName)
	Logger.Debugf("dbuser=%s", *DBUser)
	Logger.Debugf("dbpasswd=%s", *DBPassword)
	Logger.Debugf("dbsslmode=%s", *DBSSLMode)	
}
