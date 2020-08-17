// dhcp-batcher
// e-mail andy@Sonar.software if you have questions
// not for prod use, poc only.

package main

func main() {

	_ = make([]byte, 1073741824)	// chill the garbage collector out (1 gb of memory ballast)


	initBasicLogging()				// some logging items are in the config load -- this puts a basic
	                        		// logging facility together to keep things visually appealing
	if err := initConfig(); err != nil {
		return
	}

	initLogging()					// apply the logging stuff that's loaded by initConfig() (mode, format flags etc.)

	if err := checkConfig(); err != nil {
		logger.Error("An error was encountered, try 'sonarproxybatcher --help' for more info\n")
		logger.Warn(err.Error())
		return
	}



	// start the scheduler
	batchTable.initTable()
	batcherSchedulerSignal := make(chan bool)
	go batchTable.RunBatchScheduler(batcherSchedulerSignal)

	switch options.OperationMode {
	case "batch":
		logger.Info("sonarproxybatcher mode = batch")
		startBatchModeServer(batcherSchedulerSignal)
	case "proxy":
		logger.Info("sonarproxybatcher mode = proxy")
		startDHCPProxy(batcherSchedulerSignal)
	default:
		logger.Info("sonarproxybatcher mode = ?, exit")
	}
	return
}
