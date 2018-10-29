package config

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
	"github.com/vharitonsky/iniflags"	
)

var (
	// Get command options
	Addr = flag.String("Addr", "0.0.0.0:3000", "server's port. Example: 0.0.0.0:3000")
	Timeout = flag.Duration("timeout", time.Second * 15, "the duration for which the server wait for existing connections to finish - e.g. 15s or 1m")
	ListenAdmin = flag.Bool("ListenAdmin", false, "Activate the playlist administration service")
	Origin = flag.String("Origin", "*", "Source from which the request to the service is allowed. Example: 0.0.0.0:4200")

	debugLevel = flag.String("debugLevel", "info", "debug level: debug, info, warn, error, dpanic, panic, fatal. Example: error")
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
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
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
	
	Logger.Infof("addr=%s", *Addr)
	Logger.Infof("timeout=%s", *Timeout)
	Logger.Infof("ListenAdmin=%v", *ListenAdmin)
	Logger.Infof("Origin=%v", *Origin)

	Logger.Debugf("dbhost=%s", *DBHost)
	Logger.Debugf("dbport=%s", *DBPort)	
	Logger.Debugf("dbname=%s", *DBName)
	Logger.Debugf("dbuser=%s", *DBUser)
	Logger.Debugf("dbpasswd=%s", *DBPassword)
	Logger.Debugf("dbsslmode=%s", *DBSSLMode)
}
