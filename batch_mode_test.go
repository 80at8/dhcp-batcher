package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"testing"
)


func init() {
	fmt.Printf("Initializing batch_mode_test.go\n")
	logger.SetLevel(logrus.DebugLevel)

	st := 0
	key := "test"
	inst := "test.sonar.software"
	user := "test"
	pass := "test"
	mockRouter := "1.2.3.4"

	batchOptions.batchSchedulerCycleTime = &st
	batchOptions.sonarAPIKey = &key
	batchOptions.sonarAPIUsername = &user
	batchOptions.sonarInstanceName = &inst
	batchOptions.batchEndpointUsername = &user
	batchOptions.batchEndpointPassword = &pass
	batchOptions.batchEndpointRouterIPList = &mockRouter


	batchTable.initializeTable()
	batchTable.rwTableMutex.Lock()
	for k := range batchTable.entry {
		delete(batchTable.entry,k)
	}
	batchTable.rwTableMutex.Unlock()


}



type handlerTest struct {
	user string
	pass string
	URI string
	status int
	routerIP string
}


func TestBatchModeEndpointRouter(t *testing.T) {

	logger.Debug("TestBatchModeEndpointRouter")
	logger.Println("*************************************************************************************")

	user := "test"
	pass := "test"
	mockRouter := "1.2.3.4"

	initializeBatchConfiguration()
	batchTable.initializeTable()

	batchOptions.batchEndpointUsername = &user
	batchOptions.batchEndpointPassword = &pass
	batchOptions.batchEndpointRouterIPList = &mockRouter

	var th [10]handlerTest

	// good requests
	th[0].URI = "/api/dhcp_assignments?ip_address=192.168.1.10&leased_mac=AA:BB:CC:DD:EE:F0&expired=0&remote_id=test1"
	th[0].user = "test"
	th[0].pass = "test"
	th[0].status = http.StatusOK
	th[0].routerIP = mockRouter

	th[1].URI = "/api/dhcp_assignments?ip_address=192.168.1.20&leased_mac=AA:BB:CC:DD:EE:F1&expired=1&remote_id=test2"
	th[1].user = "test"
	th[1].pass = "test"
	th[1].status = http.StatusOK
	th[1].routerIP = mockRouter

	th[2].URI = "/api/dhcp_assignments?ip_address=192.168.1.30&leased_mac=AA:BB:CC:DD:EE:F2&expired=0"
	th[2].user = "test"
	th[2].pass = "test"
	th[2].status = http.StatusOK
	th[2].routerIP =mockRouter

	th[3].URI = "/api/dhcp_assignments?ip_address=192.168.1.40&leased_mac=AA:BB:CC:DD:EE:F0&expired=1"
	th[3].user = "test"
	th[3].pass = "test"
	th[3].status = http.StatusOK
	th[3].routerIP = mockRouter

	// bad requests

	// malformed IP address
	th[4].URI = "/api/dhcp_assignments?ip_address=192.168.1.&leased_mac=AA:BB:CC:DD:EE:F1&expired=0&remote_id=test5"
	th[4].user = "test"
	th[4].pass = "test"
	th[4].status = http.StatusBadRequest
	th[4].routerIP = mockRouter
	// malformed MAC address
	th[5].URI = "/api/dhcp_assignments?ip_address=192.168.1.60&leased_mac=AA:BB:CC:DD:EE&expired=0&remote_id=test6"
	th[5].user = "test"
	th[5].pass = "test"
	th[5].status = http.StatusBadRequest
	th[5].routerIP =mockRouter

	// missing expired
	th[6].URI = "/api/dhcp_assignments?ip_address=192.168.1.70&leased_mac=AA:BB:CC:DD:EE:F2&remote_id=test7"
	th[6].user = "test"
	th[6].pass = "test"
	th[6].status = http.StatusBadRequest
	th[6].routerIP = mockRouter

	longStringOne,longStringTwo :=	func() (string,string) {
		s := ""
		for x := 0; x < 1000; x++ {
			s += "*"
		}
		return "/api/dhcp_assignments?ip_address=192.168.1.10&leased_mac=AA:BB:CC:DD:EE:F0&expired=1&remote_id="+s,"/api/dhcp_assignments?ip_address=192.168.1.10&leased_mac=AA:BB:CC:DD:EE:FF&expired=0&remote_id="+s
	}()

	// remoteID is too long, when trying to expire an entry
	th[7].URI = longStringOne
	th[7].user = "test"
	th[7].pass = "test"
	th[7].status = http.StatusBadRequest
	th[7].routerIP = mockRouter

	// remoteID is too long, when trying to add a new entry
	th[8].URI = longStringTwo
	th[8].user = "test"
	th[8].pass = "test"
	th[8].status = http.StatusBadRequest
	th[8].routerIP = mockRouter

	// auth failure (user name and password mismatch)
	th[8].URI = "/api/dhcp_assignments?ip_address=192.168.1.10&leased_mac=AA:BB:CC:DD:EE:F1&expired=0&remote_id=test8"
	th[8].user = "fail"
	th[8].pass = "fail"
	th[8].status = http.StatusUnauthorized
	th[8].routerIP = mockRouter

	// unabe to parse router IP. (someh
	th[9].URI = "/api/dhcp_assignments?ip_address=192.168.1.10&leased_mac=AA:BB:CC:DD:EE:F2&expired=0&remote_id=test9"
	th[9].user = "test"
	th[9].pass = "test"
	th[9].status = http.StatusBadRequest
	th[9].routerIP = "5.5.5"

	for k := range th {

		x := httptest.NewRequest("GET",th[k].URI,nil)
		x.SetBasicAuth(th[k].user,th[k].pass)
		rr := httptest.NewRecorder()

		if k == 9 {
			x.RemoteAddr = th[k].routerIP
		}
		handler := http.HandlerFunc(BatchModeEndpointRouter)
		handler.ServeHTTP(rr, x)

		if status := rr.Code; status != th[k].status {
			t.Errorf("%v - handler returned wrong status code: got %v want %v", k,status, th[k].status)
		}
	}
	logger.Println("*************************************************************************************")
	logger.Println()
}



