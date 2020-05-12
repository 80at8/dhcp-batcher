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
	"strings"
	"time"
)

var batchProxyOptions batchConfig

type batchConfig struct {
	DHCPOperationMode           *string
	clientFacingInterface       *string
	dhcpFacingInterface         *string
	dhcpServersIP               *string
	proxyGIAddr                 *string
	isTLSEnabled                *bool
	batchEndpointTLSKey         *string
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
	sonarAPIUsername            *string
	sonarAPIKey                 *string
	sonarInstanceName           *string
}

func initializeBatchProxyConfiguration() {
	batchProxyOptions.DHCPOperationMode = flag.String("app_mode", "batch", "DHCP operation mode [ batch | proxy ]")
	batchProxyOptions.clientFacingInterface = flag.String("proxy_in", "eth0", "Interface to listen for DHCPv4/BOOTP queries on")
	batchProxyOptions.dhcpFacingInterface = flag.String("proxy_out", "eth1", "Outgoing interface for DHCP server")
	batchProxyOptions.dhcpServersIP = flag.String("proxy_dhcp_ips", "", "IP addresses of the DHCP servers [ IP1 , IP2 ]")
	batchProxyOptions.proxyGIAddr = flag.String("proxy_giaddr", "", "Required ip address (of outgoing interface and to be used as GIADDR")
	batchProxyOptions.isTLSEnabled = flag.Bool("batch_use_tls", false, "Enable TLS [ true | false ]")
	batchProxyOptions.batchEndpointTLSKey = flag.String("batch_tls_key", "/opt/sonar/dhcp-batcher/tls/dhcp-batcher.key", "Path to TLS private key")
	batchProxyOptions.batchEndpointTLSCert = flag.String("batch_tls_cert", "/opt/sonar/dhcp-batcher/tls/dhcp-batcher.crt", "Path to TLS public certificate")
	batchProxyOptions.batchEndpointUsername = flag.String("batch_username", "", "Username for batch endpoint authentication (minimum 5 characters)")
	batchProxyOptions.batchEndpointPassword = flag.String("batch_password", "", "Password for batch endpoint authentication (minimum 16 characters)")
	batchProxyOptions.batchEndpointServerIP = flag.String("batch_ip", "127.0.0.1", "Local IP to bind DHCP batching requests to")
	batchProxyOptions.batchEndpointHTTPServerPort = flag.String("batch_http_port", "80", "HTTP port to listen for dhcp batcher requests on, or redirect to TLS")
	batchProxyOptions.batchEndpointTLSServerPort = flag.String("batch_tls_port", "443", "TLS port to listen for dhcp batcher requests on")
	batchProxyOptions.batchEndpointRouterIPList = flag.String("batch_routers", "", "IP (or comma separated list of IPs) of the router(s) that will be sending DHCP entries to the batcher [\"a.b.c.d\" || \"a.b.c.d, ..., w.x.y.x\"]")
	batchProxyOptions.batchEndpointLoggingMode = flag.String("batch_logging_mode", "info", "Batch endpoint logging Level [ none | info | warn | debug ]")
	batchProxyOptions.batchEndpointLoggingFormat = flag.String("batch_logging_format", "text", "Batch endpoint logging format [ text | json ]")
	batchProxyOptions.batchEndpointLoggingOutput = flag.String("batch_logging_path", "/opt/sonar/dhcp-batcher/logs/dhcpbatcher.log", "Batch endpoint logging output [ path | \"console\"]")
	batchProxyOptions.batchSchedulerCycleTime = flag.Int("batch_cycle_time", 5, "Batch scheduler cycle time (in minutes), set to 0 to enable near-realtime batching (15 seconds)")
	batchProxyOptions.sonarVersion = flag.Int("sonar_version", 2, "Sonar version batcher will report to, [ 1 | 2 ]")
	batchProxyOptions.sonarAPIKey = flag.String("sonar_api_key", "", "V1 Sonar password or V2 Sonar bearer token")
	batchProxyOptions.sonarAPIUsername = flag.String("sonar_api_username", "", "V1 Sonar username")
	batchProxyOptions.sonarInstanceName = flag.String("sonar_instance", "", "V1 or V2 Sonar instance name (use FQDN e.g: example.sonar.software)")

	flag.Parse()

}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// checkBatchProxyConfiguration(): "what is my purpose"
// "check the batch configuration"
// checkBatchProxyConfiguration(): "what is my purpose"
// "you check the batch configuration"
// checkBatchProxyConfiguration(): omg.. :(
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func checkBatchProxyConfiguration() error {

	if *batchProxyOptions.DHCPOperationMode == "batch" {

		if *batchProxyOptions.isTLSEnabled {
			if _, err := os.Stat(*batchProxyOptions.batchEndpointTLSKey); err != nil {
				return errors.New("(--batch_tls_key) TLS key not found")
			}
			if _, err := os.Stat(*batchProxyOptions.batchEndpointTLSCert); err != nil {
				return errors.New("(--batch_tls_cert) TLS cert not found")
			}
			if _, err := strconv.Atoi(*batchProxyOptions.batchEndpointTLSServerPort); err != nil {
				return errors.New("(--batch_tls_port) TLS port not an integer")
			}
		}

		if len(*batchProxyOptions.batchEndpointUsername) < 5 {
			return errors.New("(--batch_username) username must be 5 or more characters")
		}

		if len(*batchProxyOptions.batchEndpointPassword) < 16 {
			return errors.New("(--batch_password) must be 16 or more characters")
		}

		if x := net.ParseIP(*batchProxyOptions.batchEndpointServerIP); x == nil {
			return errors.New("(--batch_ip) unable to parse server IP")
		}
	}

	if *batchProxyOptions.DHCPOperationMode == "proxy" {

		if x := net.ParseIP(*batchProxyOptions.proxyGIAddr); x == nil{
			return errors.New("(--proxy_giaddr) unable to parse proxy gateway IP")
		}

		servers := strings.Fields(*batchProxyOptions.dhcpServersIP)
		if servers == nil {
			return errors.New("(--proxy_dhcp_ips) you need to specify the IP's of the dhcp servers to proxy requests to")
		}

		for _, s := range servers {
			if x := net.ParseIP(s); x == nil {
				return errors.New("(--proxy_dhcp_ips) unable to parse dhcp server IP's")
			}

		}
	}

	// TODO -- find out if there is a fixed width API bearer token.
	if *batchProxyOptions.sonarVersion < 1 && *batchProxyOptions.sonarVersion > 2 {
		return errors.New("(--sonar_version) version must be one or two")
	}

	if len(*batchProxyOptions.sonarAPIUsername) > 256 {
		return errors.New("(--sonar_api_username) you're username is blank or greater than 256 characters")
	}

	if *batchProxyOptions.sonarAPIUsername == "" {
		return errors.New("(--sonar_api_username) you're username can't be blank")
	}

	if len(*batchProxyOptions.sonarAPIKey) > 256 {
		return errors.New("(--sonar_api_key) you're API key is greater than 256 characters")
	}

	if *batchProxyOptions.sonarAPIKey == "" {
		return errors.New("(--sonar_api_key) you're API key can't be blank")
	}

	if len(*batchProxyOptions.sonarInstanceName) > 256 {
		return errors.New("(--sonar_instance) you're instance FQDN or greater than 256 characters")
	}

	if *batchProxyOptions.sonarInstanceName == "" {
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
	if *batchProxyOptions.isTLSEnabled == true {
		// http redirect
		redirectConfig = http.Server{
			Addr: *batchProxyOptions.batchEndpointServerIP + ":" + *batchProxyOptions.batchEndpointHTTPServerPort,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Connection", "close")
				url := "https://" + *batchProxyOptions.batchEndpointServerIP + ":" + *batchProxyOptions.batchEndpointTLSServerPort + req.URL.String()
				http.Redirect(w, req, url, http.StatusMovedPermanently)
			}),
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		// tls endpoint
		endpointConfig = http.Server{
			Addr:              *batchProxyOptions.batchEndpointServerIP + ":" + *batchProxyOptions.batchEndpointTLSServerPort,
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
			Addr:              *batchProxyOptions.batchEndpointServerIP + ":" + *batchProxyOptions.batchEndpointHTTPServerPort,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		return redirectConfig, http.Server{}, nil
	}
	return http.Server{Handler: nil}, http.Server{Handler: nil}, errors.New("configBatchModeServers(): unable to populate structs\n")
}
