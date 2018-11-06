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
	debugLevel = flag.String("debugLevel", "info", "debug level: debug, info, warn, error, dpanic, panic, fatal. Example: -dlevel error")
	Log = flag.String("Log", "collector.log", "log files")
	LogError = flag.String("LogError", "collector_error.log", "log files")

	FileSecret = flag.String("fileToken", "client_secret.json", "client secret file")
	CredentialFile = flag.String("fileCredential", "yotubemetric_credential.json", "client credential file")
	
	Timeout = flag.Duration("timeout", time.Second * 15, "the duration for which the server wait for existing connections to finish - e.g. 15s or 1m")

	PeriodPlayList = flag.Duration("periodPlayList", time.Second * 600, "the frequency of checking change's play list - e.g. 600s or 1m")
	PeriodVideo = flag.Duration("periodVideo", time.Second * 60, "the frequency of checking a new video - e.g. 600s or 5m")
	PeriodMeter = flag.Duration("periodMetric", time.Second * 60, "the frequency of checking video meter - e.g. 60s or 1m")
	PeriodCount = flag.Duration("periodSaveMetricIdle", time.Hour * 1, "the frequency of preservation of metrics if they do not change - e.g. 1h or 30m")
	PeriodDeleted = flag.Duration("periodFinalDeletion", time.Hour * 24, "the frequency of delete deactivated playlist and video - e.g. 60s or 1m")
	PeriodСollection = flag.Duration("periodCollect", time.Hour * 24 * 14, "the collection period video statistics from the date and time that the video was uploaded- e.g. 336h")
	
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
	Logger = logger.Sugar()
	
	Logger.Warnf("debug level=%v", atomicLevel)
	Logger.Warnf("Log=%s", Log)
	Logger.Warnf("LogError=%s", LogError)

	Logger.Infof("fileSecret=%v", *FileSecret)
	Logger.Infof("timeout=%s", *Timeout)
	
	Logger.Infof("PeriodPlayList=%v", *PeriodPlayList)
	Logger.Infof("PeriodVideo=%v", *PeriodVideo)
	Logger.Infof("PeriodMeter=%v", *PeriodMeter)
	Logger.Infof("PeriodCount=%v", *PeriodCount)
	Logger.Infof("PeriodDeleted=%v", *PeriodDeleted)
	Logger.Infof("PeriodСollection=%v", *PeriodСollection)

	Logger.Debugf("dbhost=%s", *DBHost)
	Logger.Debugf("dbport=%s", *DBPort)
	Logger.Debugf("dbname=%s", *DBName)
	Logger.Debugf("dbuser=%s", *DBUser)
	Logger.Debugf("dbpasswd=%s", *DBPassword)
	Logger.Debugf("dbsslmode=%s", *DBSSLMode)	
}
