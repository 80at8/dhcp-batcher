package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {

	_ = make([]byte, 1073741824) // chill the garbage collector out (1 gb of memory ballast)

	initializeBatchProxyConfiguration()

	if err := checkBatchProxyConfiguration(); err != nil {
		fmt.Printf("error.\n%v\n\ntry 'dhcp-batcher --help' for more options\n\n", err.Error())
		return
	}

	initializeLogging()

	// start the scheduler
	batchTable.initializeTable()
	batchSchedulerCtl := make(chan bool)
	go batchTable.RunBatchScheduler(batchSchedulerCtl)


	switch *batchProxyOptions.DHCPOperationMode {
	case "batch":
		logger.Info("dhcp-batcher started")
		startBatchModeServer(batchSchedulerCtl)

	case "proxy":
		logger.Info("dhcp-proxy started")
		servers := strings.Fields(*batchProxyOptions.dhcpServersIP)
		for _, s := range servers {
			dhcpServers = append(dhcpServers, net.ParseIP(s))
		}
		dhcpGIAddr = net.ParseIP(*batchProxyOptions.proxyGIAddr)
		createRelay(*batchProxyOptions.clientFacingInterface, *batchProxyOptions.dhcpFacingInterface,batchSchedulerCtl)

	default:
		logger.Info("dhcp-proxy-batcher no switches specified.. exit")
	}
}
