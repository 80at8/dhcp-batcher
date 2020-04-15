package main

import (
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

const (
	INFO logrus.Level = iota
	WARN
	DEBUG
)

var logger = logrus.New()

func initializeLogging() {
	switch strings.ToLower(*batchOptions.batchEndpointLoggingFormat) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	case "text":
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			DisableLevelTruncation: true,
			DisableColors:          true,
			FullTimestamp:          true,
		})
	}

	switch strings.ToLower(*batchOptions.batchEndpointLoggingMode) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
		break
	case "info":
		logger.SetLevel(logrus.InfoLevel)
		break
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
		break
	case "none":
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.SetOutput(os.Stderr)
	if strings.ToLower(*batchOptions.batchEndpointLoggingOutput) != "console" {
		file, err := os.OpenFile(strings.ToLower(*batchOptions.batchEndpointLoggingOutput), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Error("logger: unable to open ", batchOptions.batchEndpointLoggingOutput)
			logger.Error(err.Error())
			logger.Error("logger: output is set to stderr")
		} else {
			logger.Out = file
		}
	}
}




func batchModeEndpointLogger(endpoint_uri string, condition string, routerIP string, rawquery string, err error) {
	if err != nil {
		logger.Println()
		logger.Warn(endpoint_uri, ":", condition, " .. source is from ", routerIP)
		logger.Debug("uri: ", rawquery)
		logger.Debug("err: ", err.Error())
	} else {
		logger.Println()
		logger.Warn(endpoint_uri, ":", condition, " .. source is from ", routerIP)
		logger.Debug("uri: ", rawquery)
	}
	return
}
