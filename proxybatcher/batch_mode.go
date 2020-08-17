package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// batchModeEndpointRouter, called by startBatchModeServer()
//
// responsible for parsing the inbound request URI from client Routers.
//
// 1- checks request remoteAddr (router IP) against list of allowable devices (see options.go)
// 2- then routes /api/dhcp_assignments and checks parameters and formats.
// 3- finally applies the appropriate Batch command for the desired result: either an expiry or new assignment
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type endpointBatchRequest struct {
	LeasedMacAddress string `json:"leased_mac_address"`
	IPAddress        string `json:"ip_address"`
	RemoteID         string `json:"remote_id"`
	Expired          string `json:"expired"`
}

func BatchModeEndpointRouter(w http.ResponseWriter, r *http.Request) {
	endpointURI, err := url.Parse(r.RequestURI)
	var mode string

	if err != nil {
		endpointLogger("/api/dhcp_assignments", "request URI invalid", r.RemoteAddr, endpointURI.RawQuery, err, mode)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	remoteHost, _, _ := net.SplitHostPort(r.RemoteAddr)
	routerIP := net.ParseIP(remoteHost)

	if routerIP == nil {
		endpointLogger("/api/dhcp_assignments", "unable to parse router IP address from http request", r.RemoteAddr, endpointURI.RawQuery, nil, mode)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	found := false
	routerUsername := ""
	routerPassword := ""
	for _, v := range options.Batch.Routers {
		logger.Info(v.RouterIP)
		if v.RouterIP == remoteHost {
			found = true
			routerUsername = v.Username
			routerPassword = v.Password
			break
		}
	}
	if !found {
		endpointLogger("/api/dhcp_assignments", "batch attempted from unauthorized router", remoteHost, endpointURI.RawQuery, nil, mode)
		w.WriteHeader(http.StatusUnauthorized)
	}

	switch endpointURI.Path {

	case "/api/dhcp_assignments":

		username, password, ok := r.BasicAuth()

		if !ok {
			endpointLogger("/api/dhcp_assignments", "failure (unknown) from", remoteHost, endpointURI.RawQuery, err, "auth")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if routerUsername == username && routerPassword == password {

			endpointLogger("/api/dhcp_assignments", "success from", remoteHost, endpointURI.RawQuery, nil, "auth")

			var leaseInformation endpointBatchRequest
			q, err := url.ParseQuery(endpointURI.RawQuery)

			if err != nil {
				endpointLogger("/api/dhcp_assignments", "unable to parse query (get)", remoteHost, endpointURI.RawQuery, err, "get")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if strings.ToLower(r.Method) == "post" {
				mode = "post"
				postResponse, err := ioutil.ReadAll(r.Body)
				if err != nil {
					endpointLogger("/api/dhcp_assignments", "unable to read JSON bytes from request body (post)", remoteHost, endpointURI.RawQuery, err, mode)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				err = json.Unmarshal(postResponse, &leaseInformation)
				if err != nil {
					endpointLogger("/api/dhcp_assignments", "error unmarshalling JSON to leaseInformation{ .. }", remoteHost, endpointURI.RawQuery, err, mode)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

			} else {
				mode = "get"
				if _, IP_SET := q["ip_address"]; IP_SET {
					leaseInformation.IPAddress = q["ip_address"][0]
				} else {
					leaseInformation.IPAddress = ""
				}
				if _, MAC_SET := q["leased_mac_address"]; MAC_SET {
					leaseInformation.LeasedMacAddress = q["leased_mac_address"][0]
				} else {
					leaseInformation.LeasedMacAddress = ""
				}
				if _, EXPIRED_SET := q["expired"]; EXPIRED_SET {
					leaseInformation.Expired = q["expired"][0]
				} else {
					leaseInformation.Expired = ""
				}
				if _, REMOTE_ID_SET := q["remote_id"]; REMOTE_ID_SET {
					leaseInformation.RemoteID = q["remote_id"][0]
				} else {
					leaseInformation.RemoteID = ""
				}
			}

			// leased_mac sanity checks
			if leaseInformation.LeasedMacAddress == "" {
				endpointLogger("/api/dhcp_assignments", "'leased_mac_address' parameter is undefined", remoteHost, endpointURI.RawQuery, err, mode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			hostAddr, err := net.ParseMAC(leaseInformation.LeasedMacAddress)
			if err != nil {
				endpointLogger("/api/dhcp_assignments", "unable to parse 'leased_mac_address'", remoteHost, endpointURI.RawQuery, err, mode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			leaseInformation.LeasedMacAddress = hostAddr.String()

			// ip_address sanity checks
			if leaseInformation.IPAddress == "" {
				endpointLogger("/api/dhcp_assignments", "'ip_address' parameter is undefined", remoteHost, endpointURI.RawQuery, nil, mode)
				return
			}

			hostIP := net.ParseIP(leaseInformation.IPAddress)
			if hostIP == nil {
				endpointLogger("/api/dhcp_assignments", "unable to parse 'ip_address'", remoteHost, endpointURI.RawQuery, nil, mode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			leaseInformation.IPAddress = hostIP.String()

			// expired sanity checks
			if leaseInformation.Expired == "" {
				endpointLogger("/api/dhcp_assignments", "'expired' parameter is undefined", remoteHost, endpointURI.RawQuery, err, mode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			exp, err := strconv.Atoi(leaseInformation.Expired)
			if err != nil {
				endpointLogger("/api/dhcp_assignments", "non-integer 'expired' parameter", remoteHost, endpointURI.RawQuery, nil, mode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if exp < 0 || exp > 1 {
				endpointLogger("/api/dhcp_assignments", "'expired' parameter must be 0 or 1", remoteHost, endpointURI.RawQuery, err, mode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			//// remote_id sanity checks
			if !(len(leaseInformation.RemoteID) >= 0 && len(leaseInformation.RemoteID) <= 246) {
				endpointLogger("/api/dhcp_assignments", "'remote_id' parameter length exceeds 246 bytes", remoteHost, leaseInformation.RemoteID[0:80]+"..."+leaseInformation.RemoteID[len(leaseInformation.RemoteID)-20:], nil, mode)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			mac, err := net.ParseMAC(leaseInformation.LeasedMacAddress)
			ip := net.ParseIP(leaseInformation.IPAddress)

			go batchTable.UpdateBatchTable(leaseInformation.Expired, routerIP, mac, ip, leaseInformation.RemoteID)
			w.WriteHeader(http.StatusOK)
			return

		}
		endpointLogger("/api/dhcp_assignments", "failure (credentials)", remoteHost, endpointURI.RawQuery, nil, mode)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	endpointLogger("/api/dhcp_assignments", "unknown endpoint", remoteHost, endpointURI.RawQuery, nil, mode)
	w.WriteHeader(http.StatusBadRequest)
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// startBatchModeServer, called by main()
//
// responsible for loading TLS configuration and server parameters into the endpoint listeners, also assigns the
// Batch handler endpoint to the servers. Server will either spin up single http instance, or an http listener as
// a redirect along with a TLS listener for batching. ridirect is automatically provisioned. routines are concurrent
// and will run until stop signal is sent.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func startBatchModeServer(ctl chan bool) {
	logger.SetOutput(os.Stderr)
	logger.Info("starting batcher...https://test.test.test")

	// load endpoints
	var TLSConfig tls.Config
	if options.Batch.IsTLSEnabled {
		TLSConfig = configBatchModeTLS()
		logger.Info("batcher: TLS + HTTP redirect configuration loaded")
	} else {
		logger.Info("batcher: HTTP configuration loaded")
	}

	redirectServer, endpointServer, err := configBatchModeServers(&TLSConfig)
	if err != nil {
		logger.Error("batcher: ", err.Error())
	} else {
		if options.Batch.IsTLSEnabled {
			logger.Info("batcher: TLS + HTTP redirect endpoints configured")
		} else {
			logger.Info("batcher: HTTP endpoint configured")
		}
	}

	// assign handler
	endpointServer.Handler = http.HandlerFunc(BatchModeEndpointRouter)

	if endpointServer.Handler != nil {
		logger.Info("batcher: handler attached")
	} else {
		logger.Error("batcher: handler has experienced an error")
		return
	}

	// start endpoints
	if options.Batch.IsTLSEnabled {
		logger.Info("batcher: starting TLS + HTTP redirect endpoint servers")

		go func() {

			if err := redirectServer.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					logger.Debug("batcher: HTTP redirect endpoint closed")
					logger.Debug("batcher: ", err.Error())
					return
				}
				logger.Error("batcher: HTTP redirector error")
				logger.Error("batcher: ", err.Error())
			}

		}()

		go func() {

			if err := endpointServer.ListenAndServeTLS(options.Batch.TlsCert, options.Batch.TlsKey); err != nil {
				if err == http.ErrServerClosed {
					logger.Debug("batcher: TLS endpoint closed")
					logger.Debug("batcher: ", err.Error())
					return
				}
				logger.Error("batcher: TLS endpoint error)")
				logger.Error("batcher: ", err.Error())
			}

		}()

	} else {

		logger.Warn("batcher: starting HTTP endpoint server [highly recommended you use TLS!]")
		go func() {

			if err := endpointServer.ListenAndServe(); err != nil {
				if err == http.ErrServerClosed {
					logger.Debug("batcher: HTTP endpoint closed")
					logger.Debug("batcher: ", err.Error())
					return
				}
				logger.Debug("batcher: HTTP endpoint error")
				logger.Error("batcher: ", err.Error())
			}

		}()

	}

	// listen for stop signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	// true, exit batchScheduler
	ctl <- true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := redirectServer.Shutdown(ctx); err != nil {
		logger.Error("batcher: HTTP redirect endpoint shutdown error")
		logger.Error("batcher: ", err.Error())
	}
	if err := endpointServer.Shutdown(ctx); err != nil {
		logger.Error("batcher: TLS/HTTP endpoint error")
		logger.Error("batcher: ", err.Error())
	}

	logger.Println()
	logger.Info("batcher: exit..")
	return
}
