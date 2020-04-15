package main

// TODO -- convert this to method based
// TODO -- implement conf file

import (
	"crypto/tls"
	"errors"
	"flag"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var batchOptions batchConfig

type batchConfig struct {
	DHCPOperationMode           *string
	isTLSEnabled                *bool
	batchEndpointTLSKey         *string // protection pages or memguard?
	batchEndpointTLSCert        *string
	batchEndpointUsername       *string
	batchEndpointPassword       *string
	batchEndpointServerIP       *string
	batchEndpointHTTPServerPort *string
	batchEndpointTLSServerPort  *string
	batchEndpointRouterIPList   *string
	batchEndpointLoggingMode    *string
	batchEndpointLoggingFormat  *string
	batchEndpointLoggingOutput  *string
	batchSchedulerCycleTime     *int
	sonarVersion                *int
	sonarAPIUsername *string
	sonarAPIKey                 *string
	sonarInstanceName           *string
}



func initializeBatchConfiguration() {
	batchOptions.DHCPOperationMode = flag.String("dhcp_mode", "batch", "DHCP operation mode [ batch | proxy | batchproxy ]")
	batchOptions.isTLSEnabled = flag.Bool("batch_use_tls", false, "Enable TLS [ true | false ]")
	batchOptions.batchEndpointTLSKey = flag.String("batch_tls_key", "/opt/sonar/dhcp-batcher/tls/dhcp-batcher.key", "Path to TLS private key")
	batchOptions.batchEndpointTLSCert = flag.String("batch_tls_cert", "/opt/sonar/dhcp-batcher/tls/dhcp-batcher.crt", "Path to TLS public certificate")
	batchOptions.batchEndpointUsername = flag.String("batch_username", "", "Username for batch endpoint authentication (minimum 5 characters)")
	batchOptions.batchEndpointPassword = flag.String("batch_password", "", "Password for batch endpoint authentication (minimum 16 characters)")
	batchOptions.batchEndpointServerIP = flag.String("batch_ip", "127.0.0.1", "Local IP to bind DHCP batching requests to")
	batchOptions.batchEndpointHTTPServerPort = flag.String("batch_http_port", "80", "HTTP port to listen for dhcp batcher requests on, or redirect to TLS")
	batchOptions.batchEndpointTLSServerPort = flag.String("batch_tls_port", "443", "TLS port to listen for dhcp batcher requests on")
	batchOptions.batchEndpointRouterIPList = flag.String("batch_routers", "", "IP (or comma separated list of IPs) of the router(s) that will be sending DHCP entries to the batcher [\"a.b.c.d\" || \"a.b.c.d, ..., w.x.y.x\"]")
	batchOptions.batchEndpointLoggingMode = flag.String("batch_logging_mode", "none", "Batch endpoint logging Level [ none | info | warn | debug ]")
	batchOptions.batchEndpointLoggingFormat = flag.String("batch_logging_format", "text", "Batch endpoint logging format [ text | json ]")
	batchOptions.batchEndpointLoggingOutput = flag.String("batch_logging_path", "/opt/sonar/dhcp-batcher/logs/dhcpbatcher.log", "Batch endpoint logging output [ path | \"console\"]")
	batchOptions.batchSchedulerCycleTime = flag.Int("batch_cycle_time", 5, "Batch scheduler cycle time (in minutes), set to 0 to enable near-realtime batching (15 seconds)")
	batchOptions.sonarVersion = flag.Int("sonar_version", 2, "Sonar version batcher will report to, [ 1 | 2 ]")
	batchOptions.sonarAPIKey = flag.String("sonar_api_key", "", "V1 Sonar password or V2 Sonar bearer token")
	batchOptions.sonarAPIUsername = flag.String("sonar_api_username","", "V1 Sonar username")
	batchOptions.sonarInstanceName = flag.String("sonar_instance", "", "V1 or V2 Sonar instance name (use FQDN e.g: example.sonar.software)")

	flag.Parse()

}


////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// checkBatchConfiguration(): "what is my purpose"
// "check the batch configuration"
// checkBatchConfiguration(): "what is my purpose"
// "you check the batch configuration"
// checkBatchConfiguration(): omg.. :(
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func checkBatchConfiguration() error {

	if *batchOptions.isTLSEnabled {
		if _, err := os.Stat(*batchOptions.batchEndpointTLSKey); err != nil {
			return errors.New("(--batch_tls_key) TLS key not found")
		}
		if _, err := os.Stat(*batchOptions.batchEndpointTLSCert); err != nil {
			return errors.New("(--batch_tls_cert) TLS cert not found")
		}
		if _,err := strconv.Atoi(*batchOptions.batchEndpointTLSServerPort); err != nil {
			return errors.New("(--batch_tls_port) TLS port not an integer")
		}
	}

	if len(*batchOptions.batchEndpointUsername) < 5 {
		return errors.New("(--batch_username) username must be 5 or more characters")
	}

	if len(*batchOptions.batchEndpointPassword) < 16 {
		return errors.New("(--batch_password) must be 16 or more characters")
	}

	if x := net.ParseIP(*batchOptions.batchEndpointServerIP); x == nil {
		return errors.New("(--batch_ip) unable to parse server IP")
	}

	// TODO -- find out if there is a fixed width API bearer token.
	if *batchOptions.sonarVersion < 1 && *batchOptions.sonarVersion > 2 {
		return errors.New("(--sonar_version) version must be one or two")
	}

	if len(*batchOptions.sonarAPIUsername) > 256  {
		return errors.New("(--sonar_api_username) you're username is blank or greater than 256 characters")
	}

	if *batchOptions.sonarAPIUsername == ""  {
		return errors.New("(--sonar_api_username) you're username can't be blank")
	}

	if len(*batchOptions.sonarAPIKey) > 256  {
		return errors.New("(--sonar_api_key) you're API key is greater than 256 characters")
	}

	if *batchOptions.sonarAPIKey == ""  {
		return errors.New("(--sonar_api_key) you're API key can't be blank")
	}

	if len(*batchOptions.sonarInstanceName) > 256  {
		return errors.New("(--sonar_instance) you're instance FQDN or greater than 256 characters")
	}

	if *batchOptions.sonarInstanceName == "" {
		return errors.New("(--sonar_instance) you're instance FQDN can't be blank")
	}

	return nil
}










////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// configBatchModeTLS(), called by startBatchModeServer()
//
// provides TLS configuration for TLS endpoint server, constructed as an independent function to allow more granular
// configuration of the TLS options, as many older device firmwares might require tailoring ciphersuites to be TLS
// compatible.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func configBatchModeTLS() tls.Config {

	// TODO tailor TLS CipherSuites.
	// see https://blog.cloudflare.com/exposing-go-on-the-internet/ for a good read about timeouts.

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
	if *batchOptions.isTLSEnabled == true {
		// http redirect
		redirectConfig = http.Server{
			Addr: *batchOptions.batchEndpointServerIP + ":" + *batchOptions.batchEndpointHTTPServerPort,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Connection", "close")
				url := "https://" + *batchOptions.batchEndpointServerIP + ":" + *batchOptions.batchEndpointTLSServerPort + req.URL.String()
				http.Redirect(w, req, url, http.StatusMovedPermanently)
			}),
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		// tls endpoint
		endpointConfig = http.Server{
			Addr:              *batchOptions.batchEndpointServerIP + ":" + *batchOptions.batchEndpointTLSServerPort,
			TLSConfig:         TLSConfig,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		return redirectConfig, endpointConfig, nil
	} else {
		// http endpoint
		endpointConfig = http.Server{
			Addr:              *batchOptions.batchEndpointServerIP + ":" + *batchOptions.batchEndpointHTTPServerPort,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		return redirectConfig, http.Server{}, nil
	}
	return http.Server{Handler:nil}, http.Server{Handler:nil}, errors.New("configBatchModeServers(): unable to populate structs\n")
}
