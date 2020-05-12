package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// batchModeEndpointRouter, called by startBatchModeServer()
//
// responsible for parsing the inbound request URI from client routers.
//
// 1- checks request remoteAddr (router IP) against list of allowable devices (see batch_proxy_options.go)
// 2- then routes /api/dhcp_assignments and checks parameters and formats.
// 3- finally applies the appropriate batch command for the desired result: either an expiry or new assignment
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func BatchModeEndpointRouter(w http.ResponseWriter, r *http.Request) {
	endpointURI, err := url.Parse(r.RequestURI)

	if err != nil {
		batchModeEndpointLogger("/api/dhcp_assignments","request URI invalid", r.RemoteAddr, endpointURI.RawQuery,err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	x, _, _ := net.SplitHostPort(r.RemoteAddr)
	routerIP := net.ParseIP(x)

	if routerIP == nil {
		batchModeEndpointLogger("/api/dhcp_assignments","unable to parse router IP address from http request", r.RemoteAddr, endpointURI.RawQuery,nil)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO check to see if routerIP is in list of allowed routers.

	switch endpointURI.Path {

	case "/api/dhcp_assignments":

		username,password,ok := r.BasicAuth()

		if !ok {
			batchModeEndpointLogger("/api/dhcp_assignments","auth failure", r.RemoteAddr, endpointURI.RawQuery,err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if *batchProxyOptions.batchEndpointUsername == username && *batchProxyOptions.batchEndpointPassword == password {

			q, err := url.ParseQuery(endpointURI.RawQuery)

			if err != nil {
				batchModeEndpointLogger("/api/dhcp_assignments","unable to parse query", r.RemoteAddr, endpointURI.RawQuery,err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var expired string
			if _, e := q["expired"]; e {
				_, err := strconv.Atoi(q["expired"][0])
				expired = q["expired"][0]
				if err != nil {
					batchModeEndpointLogger("/api/dhcp_assignments","non-integer 'expired' parameter", r.RemoteAddr, endpointURI.RawQuery,err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			} else {
				batchModeEndpointLogger("/api/dhcp_assignments","'expired' parameter is undefined", r.RemoteAddr, endpointURI.RawQuery,err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var hostAddr net.HardwareAddr
			if _, m := q["leased_mac"]; m {
				hostAddr, err = net.ParseMAC(q["leased_mac"][0])
				if err != nil {
					batchModeEndpointLogger("/api/dhcp_assignments","unable to parse 'leased_mac'", r.RemoteAddr, endpointURI.RawQuery,err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			} else {
				batchModeEndpointLogger("/api/dhcp_assignments","'leased_mac' parameter is undefined", r.RemoteAddr, endpointURI.RawQuery,err)
				return
			}

			var hostIP net.IP
			if _, i := q["ip_address"]; i {
				hostIP = net.ParseIP(q["ip_address"][0])
				if hostIP == nil {
					batchModeEndpointLogger("/api/dhcp_assignments","unable to parse 'ip_address'", r.RemoteAddr, endpointURI.RawQuery,nil)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			} else {
				batchModeEndpointLogger("/api/dhcp_assignments","'ip_address' parameter is undefined", r.RemoteAddr, endpointURI.RawQuery,nil)
				return
			}

			var remoteID string
			if _, rid := q["remote_id"]; rid {
				if len(q["remote_id"][0]) <= 246 {
					// max 246 characters.
					remoteID = q["remote_id"][0]
				} else {
					batchModeEndpointLogger("/api/dhcp_assignments","'remote_id' parameter length exceeds 246 bytes", r.RemoteAddr, endpointURI.RawQuery[0:80] + "..." + endpointURI.RawQuery[len(endpointURI.RawQuery)-20:],nil)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}

			go batchTable.UpdateBatchTable(expired, routerIP, hostAddr, hostIP, remoteID)
			w.WriteHeader(http.StatusOK)
			return
		}
		batchModeEndpointLogger("/api/dhcp_assignments","authenication failure", r.RemoteAddr, endpointURI.RawQuery,nil)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}


	batchModeEndpointLogger("/api/dhcp_assignments","unknown endpoint", r.RemoteAddr, endpointURI.RawQuery,nil)
	w.WriteHeader(http.StatusBadRequest)
	return
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// startBatchModeServer, called by main()
//
// responsible for loading TLS configuration and server parameters into the endpoint listeners, also assigns the
// batch handler endpoint to the servers. Server will either spin up single http instance, or an http listener as
// a redirect along with a TLS listener for batching. ridirect is automatically provisioned. routines are concurrent
// and will run until stop signal is sent.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func startBatchModeServer(ctl chan bool) {
	logger.SetOutput(os.Stderr)
	logger.Info("starting batcher...")

	// load endpoints
	var TLSConfig tls.Config
	if *batchProxyOptions.isTLSEnabled {
		TLSConfig = configBatchModeTLS()
		logger.Info("batcher: TLS + HTTP redirect configuration loaded")
	} else {
		logger.Info("batcher: HTTP configuration loaded")
	}

	redirectServer, endpointServer, err := configBatchModeServers(&TLSConfig)
	if err != nil {
		logger.Error("batcher: ", err.Error())
	} else {
		if *batchProxyOptions.isTLSEnabled {
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
	if *batchProxyOptions.isTLSEnabled {
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

			if err := endpointServer.ListenAndServeTLS(*batchProxyOptions.batchEndpointTLSCert, *batchProxyOptions.batchEndpointTLSKey); err != nil {
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

	logger.Info("batcher: exit..")
	return
}
