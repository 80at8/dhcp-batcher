// dhcp-batcher
// e-mail andy@sonar.software if you have questions
// not for prod use, poc only.

package main

import (
	"fmt"
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
	batcherSchedulerSignal := make(chan bool)
	go batchTable.RunBatchScheduler(batcherSchedulerSignal)


	switch *batchProxyOptions.DHCPOperationMode {
	case "batch":
		logger.Info("dhcp-batcher started")
		startBatchModeServer(batcherSchedulerSignal)
	case "proxy":
		logger.Info("dhcp-proxy started")
		startDHCPProxy(batcherSchedulerSignal)
	default:
		logger.Info("dhcp-proxy-batcher no switches specified.. exit")
	}
	return
}
