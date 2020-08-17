package main

// TODO -- convert this to method based
// TODO -- implement conf file

import (
	"crypto/tls"
	"errors"
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var options programConfig

type programConfig struct {
	OperationMode string        `yaml:"operation_mode"`
	Sonar         sonarConfig   `yaml:"sonar"`
	Batch         batchConfig   `yaml:"batch"`
	Proxy         proxyConfig   `yaml:"proxy"`
	Logging       loggingConfig `yaml:"logging"`
}

type sonarConfig struct {
	Version      int    `yaml:"sonar_version"`
	ApiUsername  string `yaml:"sonar_api_username"`
	ApiKey       string `yaml:"sonar_api_key"`
	InstanceName string `yaml:"sonar_instance"`
	BearerToken  string `yaml:"sonar_bearer_token"`
}

type batchRouterAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	RouterIP string `yaml:"router_ip"`
}

type batchConfig struct {
	IsTLSEnabled       bool              `yaml:"batch_use_tls"`
	TlsKey             string            `yaml:"batch_tls_key"`
	TlsCert            string            `yaml:"batch_tls_cert"`
	EndpointUsername   string            `yaml:"batch_username"`
	EndpointPassword   string            `yaml:"batch_password"`
	ServerIP           string            `yaml:"batch_ip"`
	HttpServerPort     string            `yaml:"batch_http_port"`
	TlsServerPort      string            `yaml:"batch_tls_port"`
	SchedulerCycleTime int               `yaml:"batch_cycle_time"`
	Routers            []batchRouterAuth `yaml:"batch_routers"`
}

type proxyConfig struct {
	UpstreamInterface   string   `yaml:"proxy_upstream_if"`
	DownstreamInterface string   `yaml:"proxy_downstream_if"`
	UpstreamServerIPs   []string `yaml:"proxy_upstream_dhcp_ips"`
	ProxyServerIP       string   `yaml:"proxy_server_ip"`
}

type loggingConfig struct {
	Mode   string `yaml:"logging_mode"`
	Format string `yaml:"logging_format"`
	Output string `yaml:"logging_output"`
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func initConfig() error {

	// call the flags first
	loggingMode := flag.String("logging_mode", "", "Logging level [ none | info | warn | debug ]")
	loggingOutput := flag.String("logging_output", "", "Batch endpoint and proxy logging output [ path | \"console\"]")
	runConfigurator := flag.Bool("configurator", false, "Run the configurator tool to configure program options for the first time")
	loadYaml := flag.Bool("loadyaml", false, "Load existing proxybatcher.yaml file into configurator")

	flag.Parse()

	configFile, err := ioutil.ReadFile("./conf/proxybatcher.yaml")


	if err != nil {
		logger.Error("unable to open ./conf/proxybatcher.yaml\n")
		logger.Error(err.Error() + "\n\n")
		logger.Error("check if file exists, permissions are correct or run with `--configurator true` to run the configurator\n\n")
		return err
	} else {
		logger.Info("./conf/proxybatcher.yaml found, opening...")
	}

	err = yaml.Unmarshal(configFile, &options)

	if err != nil {
		// unable to parse the yaml config -- so we can't run.
		// offer to create a new yaml config.
		logger.Error("error unmarshalling ./conf/proxybatcher.yaml\n")
		logger.Error(err.Error() + "\n\n")
		logger.Error("run with `--configurator true` to run the configurator\n\n")
		return err
	} else {
		logger.Info("./conf/proxybatcher.yaml successfully unmarshalled.. config loaded.")
	}

	// lets make a new configuration
	if *runConfigurator == true {
		configurator(*loadYaml)
		return errors.New("exit")
	}



	if isFlagPassed("logging_mode") {
		if *loggingMode == "none" || *loggingMode == "info" || *loggingMode == "warn" || *loggingMode == "debug" {
			options.Logging.Mode = *loggingMode
			logger.Info("logging_mode flag set, override is ", *loggingMode)
		} else {
			logger.Warn("invalid flag specified for logging_mode, use: none, info, warn, or debug")
		}
	}

	if isFlagPassed("logging_output") {
		if *loggingOutput == "console" {
			options.Logging.Output = "console"
			logger.Info("logging_output flag set, override is console")
		} else {
			if _, err := os.Stat(*loggingOutput); !os.IsNotExist(err) {
				options.Logging.Output = *loggingOutput
				logger.Info("logging_output flag set, override is ", *loggingOutput)
			} else {
				logger.Warn("invalid flag specified for logging_output, use: console or valid path to output directory")
				logger.Warn(err.Error(), ":", *loggingOutput)
			}
		}
	}

	logger.Info("loading router list")
	for _, v := range options.Batch.Routers {
		logger.Info("router: ", v.RouterIP, " username: ", v.Username, " password: hidden")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// checkConfig(): "what is my purpose"
// "check the Batch configuration"
// checkConfig(): "what is my purpose"
// "you check the Batch configuration"
// checkConfig(): omg.. :(
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func checkConfig() error {

	if strings.ToLower(options.OperationMode) == "batch" {

		if options.Batch.IsTLSEnabled {
			if _, err := os.Stat(options.Batch.TlsKey); err != nil {
				return errors.New("(batch_tls_key) TLS key not found")
			}
			if _, err := os.Stat(options.Batch.TlsCert); err != nil {
				return errors.New("(batch_tls_cert) TLS cert not found")
			}
			if _, err := strconv.Atoi(options.Batch.TlsServerPort); err != nil {
				return errors.New("(batch_tls_port) TLS port must be an integer")
			}
		}

		if len(options.Batch.EndpointUsername) < 5 {
			return errors.New("(batch_username) endpoint username must be 5 or more characters")
		}

		if len(options.Batch.EndpointPassword) < 16 {
			return errors.New("(batch_password) endpoint password must be 16 or more characters")
		}

		if x := net.ParseIP(options.Batch.ServerIP); x == nil {
			return errors.New("(batch_ip) unable to parse server IP")
		}
	}

	if strings.ToLower(options.OperationMode) == "proxy" {

		if x := net.ParseIP(options.Proxy.ProxyServerIP); x == nil {
			return errors.New("(proxy_server_ip) unable to parse the specified proxy server IP")
		}

		if len(options.Proxy.UpstreamServerIPs) == 0 {
			return errors.New("(proxy_upstream_dhcp_ips) you need to specify the IP's of the dhcp servers to forward the requests to")
		}

		for _, s := range options.Proxy.UpstreamServerIPs {
			if x := net.ParseIP(s); x == nil {
				return errors.New("(proxy_upstream_dhcp_ips) unable to parse the specified dhcp server IP's")
			}

		}
	}

	if options.Sonar.Version < 1 && options.Sonar.Version > 2 {
		return errors.New("(sonar_version) version must be [1 | 2]")
	}

	if options.Sonar.Version == 1 {

		if len(options.Sonar.ApiUsername) > 256 {
			return errors.New("(v1 sonar_api_username) your sonar_api_username is blank or greater than 256 characters")
		}

		if options.Sonar.ApiUsername == "" {
			return errors.New("(v1 sonar_api_username) your sonar_api_username can't be blank")
		}

		if len(options.Sonar.ApiKey) > 1925 {
			return errors.New("(v1 sonar_api_key) your sonar_api_key is greater than 1925 bytes")
		}

		if options.Sonar.ApiKey == "" {
			return errors.New("(v1 sonar_api_key) your sonar_api_key key can't be blank")
		}

	}

	if options.Sonar.Version == 2 {

		if len(options.Sonar.BearerToken) > 1925 {
			return errors.New("(v1 sonar_bearer_token) your sonar_bearer_token is greater than 1925 bytes")
		}

	}

	options.Sonar.InstanceName = strings.ToLower(options.Sonar.InstanceName)
	options.Sonar.InstanceName = strings.Replace(options.Sonar.InstanceName, "https://", "", 1)

	if len(options.Sonar.InstanceName) > 256 || options.Sonar.InstanceName == "" {
		return errors.New("(sonar_instance) your sonar_instance URI is blank or greater than 256 characters")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// configBatchModeTLS(), called by startBatchModeServer()
//
// provides TLS configuration for TLS endpoint server, constructed as an independent function to allow more granular
// configuration of the TLS Batch, as many older device firmwares might require tailoring ciphersuites to be TLS
// compatible.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func configBatchModeTLS() tls.Config {
	TLSConfig := tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			// tls.CurveP384,
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

			// best disabled, as they don't provide forward secrecy, but might be necessary for some clients.
			// enable at your own risk.

			// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			// tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
	return TLSConfig
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// configBatchModeServers(), called by startBatchModeServer()
//
// tls.Config is passed into this function and it gets applied to the http.Server listeners. provide either a insecure
// http endpoint, or an http redirector + TLS endpoint for secure batching. constructed as an independant function to
// allow granular configuration of timeouts and other http.server parameters.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func configBatchModeServers(TLSConfig *tls.Config) (http.Server, http.Server, error) {
	var redirectConfig, endpointConfig http.Server
	if options.Batch.IsTLSEnabled == true {
		// http redirect
		redirectConfig = http.Server{
			Addr: options.Batch.ServerIP + ":" + options.Batch.HttpServerPort,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Connection", "close")
				url := "https://" + options.Batch.ServerIP + ":" + options.Batch.TlsServerPort + req.URL.String()
				http.Redirect(w, req, url, http.StatusMovedPermanently)
			}),
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		// tls endpoint
		endpointConfig = http.Server{
			Addr:              options.Batch.ServerIP + ":" + options.Batch.TlsServerPort,
			TLSConfig:         TLSConfig,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		return redirectConfig, endpointConfig, nil
	}
	// http endpoint
	endpointConfig = http.Server{
		Addr:              options.Batch.ServerIP + ":" + options.Batch.HttpServerPort,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return http.Server{}, endpointConfig, nil
}
