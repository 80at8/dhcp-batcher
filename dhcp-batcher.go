package main

import "fmt"

func main() {

	_ = make([]byte, 1073741824) // chill the garbage collector out (1 gb of memory ballast)

	initializeBatchConfiguration()

	if err := checkBatchConfiguration(); err != nil {
		fmt.Printf("error.\n%v\n\ntry 'dhcp-batcher --help' for more options\n\n", err.Error())
		return
	}

	initializeLogging()

	logger.Info("dhcp-batcher started")

	startBatchModeServer()

}
