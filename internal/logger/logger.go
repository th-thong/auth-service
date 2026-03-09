package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
    var config zap.Config

    if os.Getenv("APP_ENV") == "production" {
        config = zap.NewProductionConfig()
        config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
        config.OutputPaths = []string{"stdout"} 
        config.ErrorOutputPaths = []string{"stderr"}
    } else {
        config = zap.NewDevelopmentConfig()
        config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
        config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
        config.OutputPaths = []string{"stdout"}
    }

    var err error
    Log, err = config.Build()
    if err != nil {
        panic(err)
    }
    
    zap.ReplaceGlobals(Log)
}