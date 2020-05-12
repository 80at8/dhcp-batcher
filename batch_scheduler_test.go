package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"testing"
)

func init() {
	fmt.Printf("Initializing batch_scheduler_test.go\n")
	logger.SetLevel(logrus.DebugLevel)

	st := 0
	key := "test"
	user := "test"
	inst := "test.sonar.software"

	batchProxyOptions.batchSchedulerCycleTime = &st
	batchProxyOptions.sonarAPIKey = &key
	batchProxyOptions.sonarAPIUsername = &user
	batchProxyOptions.sonarInstanceName = &inst

	batchTable.initializeTable()
	batchTable.rwTableMutex.Lock()
	for k := range batchTable.entry {
		delete(batchTable.entry, k)
	}
	batchTable.rwTableMutex.Unlock()

}

type schedulerUpdateTest struct {
	expired  string
	routerIP net.IP
	hostAddr net.HardwareAddr
	hostIP   net.IP
	remoteID string
}

func TestRecordTable_UpdateBatchTable(t *testing.T) {
	logger.Debug("TestRecordTable_UpdateBatchTable")
	logger.Println("*************************************************************************************")

	var ts [3]schedulerUpdateTest

	// 3 adds
	ts[0].expired = "0" // expired
	ts[0].routerIP = net.ParseIP("192.168.1.1")
	ts[0].hostAddr, _ = net.ParseMAC("AA:BB:CC:DD:EE:F0")
	ts[0].hostIP = net.ParseIP("192.168.1.100")
	ts[0].remoteID = "add1"

	ts[1].expired = "0" // expired
	ts[1].routerIP = net.ParseIP("192.168.1.1")
	ts[1].hostAddr, _ = net.ParseMAC("AA:BB:CC:DD:EE:F1")
	ts[1].hostIP = net.ParseIP("192.168.1.101")
	ts[1].remoteID = "add2"

	ts[2].expired = "0" // expired
	ts[2].routerIP = net.ParseIP("192.168.1.1")
	ts[2].hostAddr, _ = net.ParseMAC("AA:BB:CC:DD:EE:F2")
	ts[2].hostIP = net.ParseIP("192.168.1.102")
	ts[2].remoteID = "add3"

	// test additions
	for k := range ts {
		batchTable.UpdateBatchTable(ts[k].expired, ts[k].routerIP, ts[k].hostAddr, ts[k].hostIP, ts[k].remoteID)
	}

	if len(batchTable.entry) != 3 {
		t.Errorf("testing additions, expected len of %v, got len of %v\n", len(ts), len(batchTable.entry))
	}

	// test map updates
	ts[0].expired = "1"
	ts[0].remoteID = "change1"

	ts[1].expired = "1"
	ts[1].remoteID = "change2"

	ts[2].expired = "1"
	ts[2].remoteID = "change3"

	for k := range ts {
		batchTable.UpdateBatchTable(ts[k].expired, ts[k].routerIP, ts[k].hostAddr, ts[k].hostIP, ts[k].remoteID)
	}

	if len(batchTable.entry) != 3 {

		for k := range batchTable.entry {
			fmt.Printf("ex:%v mac:%v ip:%v rem:%v\n", batchTable.entry[k].Expired, batchTable.entry[k].MacAddress, batchTable.entry[k].IpAddress, batchTable.entry[k].RemoteID)
		}

		t.Errorf("testing for updates, expected len of 3, got len of %v", len(batchTable.entry))
	}

	for k := range ts {
		if ts[k].expired != "1" {
			t.Errorf("testing updates, expected expired of 1, got expired of %v", ts[k].expired)
		}
		if ts[k].remoteID != "change"+strconv.Itoa(k+1) {
			t.Errorf("testing updates, expected remoteID of %v, got remoteID of %v", "change"+strconv.Itoa(k+1), ts[k].remoteID)
		}
	}

	logger.Println("*************************************************************************************")
	logger.Println()
}

func BenchmarkRecordTable_UpdateBatchTable(b *testing.B) {
	// 3 adds

	var ts [3]schedulerUpdateTest

	ts[0].expired = "0" // expired
	ts[0].routerIP = net.ParseIP("192.168.1.1")
	ts[0].hostAddr, _ = net.ParseMAC("AA:BB:CC:DD:EE:F0")
	ts[0].hostIP = net.ParseIP("192.168.1.100")
	ts[0].remoteID = "add1"

	ts[1].expired = "0" // expired
	ts[1].routerIP = net.ParseIP("192.168.1.1")
	ts[1].hostAddr, _ = net.ParseMAC("AA:BB:CC:DD:EE:F1")
	ts[1].hostIP = net.ParseIP("192.168.1.101")
	ts[1].remoteID = "add2"

	ts[2].expired = "0" // expired
	ts[2].routerIP = net.ParseIP("192.168.1.1")
	ts[2].hostAddr, _ = net.ParseMAC("AA:BB:CC:DD:EE:F2")
	ts[2].hostIP = net.ParseIP("192.168.1.102")
	ts[2].remoteID = "add3"
		// test additions

	for x := 0; x < 2; x++ {
		batchTable.UpdateBatchTable(ts[x].expired, ts[x].routerIP, ts[x].hostAddr, ts[x].hostIP, ts[x].remoteID)
	}

}
