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


func initBasicLogging() {
	logger.SetFormatter(&logrus.TextFormatter{
		DisableLevelTruncation: true,
		DisableColors:          false,
		FullTimestamp:          true,
	})
	logrus.SetLevel(logrus.DebugLevel)
	logger.Out = os.Stdout
}




func initLogging() {
	switch strings.ToLower(options.Logging.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	case "text":
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			DisableLevelTruncation: true,
			DisableColors:          false,
			FullTimestamp:          true,
		})
	}

	switch strings.ToLower(options.Logging.Mode) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
		break
	case "info":
		logger.SetLevel(logrus.InfoLevel)
		break
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
		break
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.SetOutput(os.Stderr)
	if strings.ToLower(options.Logging.Output) != "console" {
		file, err := os.OpenFile(strings.ToLower(options.Logging.Output), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Error("logger: unable to open ", options.Logging.Output)
			logger.Error(err.Error())
			logger.Error("logger: output will be stderr")
		} else {
			logger.Out = file
		}
	}
}

func endpointLogger(endpoint_uri string, condition string, routerIP string, rawquery string, err error, mode string) {
	if mode == "get" {
		if err != nil {
			logger.Warn("get: ", endpoint_uri, ":", condition, ", source ", routerIP)
			logger.Debug("uri: ", rawquery)
			logger.Debug("err: ", err.Error())
		} else {
			logger.Info("get: ", endpoint_uri, ":", condition, ", source ", routerIP)
			logger.Debug("uri: ", rawquery)
		}
	}
	if mode == "post" {
		if err != nil {
			logger.Warn("post: ", endpoint_uri, ":", condition, ", source ", routerIP)
			logger.Debug("err: ", err.Error())
		} else {
			logger.Info("post: ", endpoint_uri, ":", condition, ", source ", routerIP)
		}
	}
	if mode == "auth" {
		if err != nil {
			logger.Warn("auth: ", endpoint_uri, ":", condition," ", routerIP)
			logger.Debug("err: ", err.Error())
		} else {
			logger.Info("auth: ", endpoint_uri, ":", condition," ", routerIP)
		}
	}
	if mode == "" {
		if err != nil {
			logger.Warn(endpoint_uri, ":", condition, ", source ", routerIP)
			logger.Debug("uri: ", rawquery)
			logger.Debug("err: ", err.Error())
		} else {
			logger.Info(endpoint_uri, ":", condition, ", source ", routerIP)
			logger.Debug("uri: ", rawquery)
		}
	}
	return
}

