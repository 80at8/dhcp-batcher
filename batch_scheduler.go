package main

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

type batchID int

// Used in
type Assignment struct {
	Expired    string `json:"expired"`
	IpAddress  string `json:"ip_address"`
	MacAddress string `json:"mac_address"`
	RemoteID   string `json:"remote_id"`
}

type recordTable struct {
	rwTableMutex  sync.Mutex
	cycleTime     time.Duration
	sonarInstance string
	sonarAPIKey   string
	sonarUser     string
	currentID     batchID
	skippedID     batchID
	entry         map[string]Assignment
}

var batchTable recordTable

func (b *recordTable) initializeTable() {
	logger.Info("initializing batch scheduler.")
	b.entry = make(map[string]Assignment)
	b.currentID = batchID(0)
	b.skippedID = batchID(0)
	b.sonarAPIKey = *batchProxyOptions.sonarAPIKey
	b.sonarUser = *batchProxyOptions.sonarAPIUsername
	b.sonarInstance = *batchProxyOptions.sonarInstanceName
	logger.Info("scheduler init: sonar instance is ", b.sonarInstance)
	if *batchProxyOptions.batchSchedulerCycleTime == 0 {
		b.cycleTime = time.Duration(15) * time.Second // realtime polling.
		logger.Warn("scheduler init: batch scheduling cycle time is set to near-realtime (15 seconds), for sonar instances with large client subnets this is can be a problem")
		logger.Warn("                consider using '--batch_cycle_time 5' (5 minutes) if you have issues")
	} else {
		b.cycleTime = time.Duration(*batchProxyOptions.batchSchedulerCycleTime) * time.Minute
		logger.Info("scheduler init: batch scheduling cycle time is set to ", b.cycleTime.String())
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// updateBatchTable, called by batchModeEndpointRouter()
//
// responsible for adding entries to the batch table.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (b *recordTable) UpdateBatchTable(expired string, routerIP net.IP, hostAddr net.HardwareAddr, hostIP net.IP, remoteID string) {
	x := Assignment{
		Expired:    expired,
		MacAddress: hostAddr.String(),
		IpAddress:  hostIP.String(),
		RemoteID:   remoteID,
	}

	// map operations aren't thread safe -- put any map changes within the mutex locks to avoid read/write
	// race conditions

	b.rwTableMutex.Lock()
	b.entry[hostAddr.String()] = x
	b.rwTableMutex.Unlock()

	if logger.GetLevel() == logrus.DebugLevel {
		logger.Debug("scheduler updater: updated record ", x.IpAddress, "[", x.MacAddress, "] expiry is ", x.Expired, " .. record updated by router with ip ", routerIP.String())
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// runBatchScheduler, called by main()
//
// responsible for scheduling batch updates to the users sonar instance
//
// 1- gets sonar instance, API key and scheduling parameters
// 2- manages adds / removals from the batch scheduling list (applies a mutex lock/unlock since list is map)
// 3- batches request to sonar, gets response.
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (b *recordTable) RunBatchScheduler(ctl chan bool) {
	logger.Info("scheduler started")
	logger.Info("press ctrl+c to terminate")

	t := time.NewTicker(b.cycleTime)

	for {
		select {
		case <-ctl:
			logger.Info("scheduler: exit..")
			return
		case <-t.C:
			logger.Info("scheduler: running scheduled batch, batch number is ", b.currentID)

			var t []Assignment

			if *batchProxyOptions.DHCPOperationMode == "proxy" {
				if len(leaseTable.entry) > 0 {

					logger.Info("scheduler: mode is proxy")
					leaseTable.mutex.Lock()
					for _, v := range leaseTable.entry {
						x := Assignment{
							Expired:    v.isExpired,
							IpAddress:  v.ip,
							MacAddress: v.mac,
							RemoteID:   v.rid,
						}
						t = append(t, x)
						//delete(leaseTable.entry, k)
					}
					//leaseTable.entry = make(map[string]lease)
					leaseTable.mutex.Unlock()

					b.currentID++
					go sendBatch(t)
				} else {
					b.skippedID++
					logger.Info("batch scheduler: proxy table is empty.. skipping (", b.skippedID, ")")
				}
			} else {
				if len(b.entry) > 0 {

					logger.Info("scheduler: mode is batch")
					// map operations aren't thread safe -- put any map changes within the mutex locks to avoid read/write
					// race conditions

					b.rwTableMutex.Lock()
					for k, v := range b.entry {
						t = append(t, v)
						delete(b.entry, k)
					}
					b.entry = make(map[string]Assignment)
					b.rwTableMutex.Unlock()

					// increment the batch number as the batch table is now cleared
					b.currentID++

					// send it off to sonar!
					go sendBatch(t)
				} else {
					b.skippedID++
					logger.Info("batch scheduler: batch table is empty.. skipping (", b.skippedID, ")")
				}
			}
		}
	}
}

func sendBatch(t []Assignment) {

	data, err := json.Marshal(map[string][]Assignment{"data": t})

	if err != nil {
		logger.Error("scheduler dispatch: error marshalling entry table to JSON")
		logger.Error(err.Error())
	}

	if logger.GetLevel() == logrus.DebugLevel {
		logger.Println()
		logger.Debug("scheduler dispatch: ---dhcp-batcher to sonar json start---")
		logger.Println()
		logger.Debug(string(data))
		logger.Println()
		logger.Debug("scheduler dispatch: ---dhcp-batcher to sonar json end---")
		logger.Println()
	}

	if *batchProxyOptions.sonarVersion == 1 {

		client := http.Client{}

		req, err := http.NewRequest("POST", "https://"+*batchProxyOptions.sonarInstanceName+"/api/v1/network/ipam/batch_dynamic_ip_assignment", bytes.NewBuffer(data))
		if err != nil {
			logger.Error("error posting to sonar instance ", *batchProxyOptions.sonarInstanceName)
			logger.Error(err.Error())
		}

		req.SetBasicAuth(*batchProxyOptions.sonarAPIUsername, *batchProxyOptions.sonarAPIKey)
		req.Header.Set("Content-Type", "application/json")

		response, err := client.Do(req)

		if err != nil {
			logger.Error("scheduler dispatch: sonar response error")
			logger.Error(err.Error())
			return
		}

		responseData, err := ioutil.ReadAll(response.Body)

		if err != nil {
			logger.Error("scheduler dispatch: unable to read response body")
			logger.Error(err.Error())
			return
		}

		if logger.GetLevel() == logrus.DebugLevel {
			logger.Println()
			logger.Debug("scheduler dispatch: ---sonar instance response start---")
			logger.Debug(string(responseData))
			logger.Debug("scheduler dispatch: ---sonar instance response end---")
			logger.Println()
		}

	}

	// add v2 endpoint code here
	if *batchProxyOptions.sonarVersion == 2 {

	}

	return
}
