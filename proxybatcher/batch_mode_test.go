package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"testing"
)

func init() {
	fmt.Printf("Initializing batch_mode_test.go\n")
	logger.SetLevel(logrus.InfoLevel)

	//st := 0
	//key := "test"
	//inst := "test.Sonar.software"
	//user := "test"
	//pass := "test"

	//options.Batch.SchedulerCycleTime = st
	//options.Sonar.ApiKey = key
	//options.Sonar.ApiUsername = user
	//options.Sonar.InstanceName = inst
	//options.Batch.EndpointUsername = user
	//options.Batch.EndpointPassword = pass

	batchTable.initTable()
	batchTable.rwTableMutex.Lock()
	for k := range batchTable.entry {
		delete(batchTable.entry, k)
	}
	batchTable.rwTableMutex.Unlock()

}

type mockGetRequest struct {
	user     string
	pass     string
	URI      string
	status   int
	routerIP string
}

type mockPostRequest struct {
	LeasedMacAddress string `json:"leased_mac_address"`
	IPAddress        string `json:"ip_address"`
	RemoteID         string `json:"remote_id"`
	Expired          string `json:"expired"`
	routerIP         string `json:"-"`
	status           int    `json:"-"`
	user             string `json:"-"`
	pass             string `json:"-"`
}

func TestBatchModeEndpointRouter(t *testing.T) {


	mockRouter := "192.0.2.1:1234"

	handler := http.HandlerFunc(BatchModeEndpointRouter)
	batchTable.initTable()

	options.Batch.Routers = []batchRouterAuth{
		{
			Username: "test",
			Password: "test",
			RouterIP: "192.0.2.1",
		},
	}

	var getTestHandler [11]mockGetRequest

	// good requests
	getTestHandler[0].URI = "/api/dhcp_assignments?ip_address=192.168.1.10&leased_mac_address=AA:BB:CC:DD:EE:F0&expired=0&remote_id=test1"
	getTestHandler[0].user = "test"
	getTestHandler[0].pass = "test"
	getTestHandler[0].status = http.StatusOK
	getTestHandler[0].routerIP = mockRouter

	getTestHandler[1].URI = "/api/dhcp_assignments?ip_address=192.168.1.20&leased_mac_address=AA:BB:CC:DD:EE:F1&expired=1&remote_id=test2"
	getTestHandler[1].user = "test"
	getTestHandler[1].pass = "test"
	getTestHandler[1].status = http.StatusOK
	getTestHandler[1].routerIP = mockRouter

	getTestHandler[2].URI = "/api/dhcp_assignments?ip_address=192.168.1.30&leased_mac_address=AA:BB:CC:DD:EE:F2&expired=0"
	getTestHandler[2].user = "test"
	getTestHandler[2].pass = "test"
	getTestHandler[2].status = http.StatusOK
	getTestHandler[2].routerIP = mockRouter

	getTestHandler[3].URI = "/api/dhcp_assignments?ip_address=192.168.1.40&leased_mac_address=AA:BB:CC:DD:EE:F0&expired=1"
	getTestHandler[3].user = "test"
	getTestHandler[3].pass = "test"
	getTestHandler[3].status = http.StatusOK
	getTestHandler[3].routerIP = mockRouter

	// bad requests

	// malformed IP address
	getTestHandler[4].URI = "/api/dhcp_assignments?ip_address=192.168.1.&leased_mac_address=AA:BB:CC:DD:EE:F1&expired=0&remote_id=test5"
	getTestHandler[4].user = "test"
	getTestHandler[4].pass = "test"
	getTestHandler[4].status = http.StatusBadRequest
	getTestHandler[4].routerIP = mockRouter

	// malformed MAC address
	getTestHandler[5].URI = "/api/dhcp_assignments?ip_address=192.168.1.60&leased_mac_address=AA:BB:CC:DD:EE&expired=0&remote_id=test6"
	getTestHandler[5].user = "test"
	getTestHandler[5].pass = "test"
	getTestHandler[5].status = http.StatusBadRequest
	getTestHandler[5].routerIP = mockRouter

	// missing expired
	getTestHandler[6].URI = "/api/dhcp_assignments?ip_address=192.168.1.70&leased_mac_address=AA:BB:CC:DD:EE:F2&remote_id=test7"
	getTestHandler[6].user = "test"
	getTestHandler[6].pass = "test"
	getTestHandler[6].status = http.StatusBadRequest
	getTestHandler[6].routerIP = mockRouter

	longStringOne, longStringTwo := func() (string, string) {
		s := ""
		for x := 0; x < 1000; x++ {
			s += "*"
		}
		return "/api/dhcp_assignments?ip_address=192.168.1.80&leased_mac_address=AA:BB:CC:DD:EE:F0&expired=1&remote_id=" + s, "/api/dhcp_assignments?ip_address=192.168.1.10&leased_mac=AA:BB:CC:DD:EE:FF&expired=0&remote_id=" + s
	}()

	// remoteID is too long, when trying to expire an entry
	getTestHandler[7].URI = longStringOne
	getTestHandler[7].user = "test"
	getTestHandler[7].pass = "test"
	getTestHandler[7].status = http.StatusBadRequest
	getTestHandler[7].routerIP = mockRouter

	// remoteID is too long, when trying to add a new entry
	getTestHandler[8].URI = longStringTwo
	getTestHandler[8].user = "test"
	getTestHandler[8].pass = "test"
	getTestHandler[8].status = http.StatusBadRequest
	getTestHandler[8].routerIP = mockRouter

	// auth failure (user name and Password mismatch)
	getTestHandler[9].URI = "/api/dhcp_assignments?ip_address=192.168.1.100&leased_mac_address=AA:BB:CC:DD:EE:F1&expired=0&remote_id=test8"
	getTestHandler[9].user = "fail"
	getTestHandler[9].pass = "fail"
	getTestHandler[9].status = http.StatusUnauthorized
	getTestHandler[9].routerIP = mockRouter

	// unable to parse router IP.
	getTestHandler[10].URI = "/api/dhcp_assignments?ip_address=192.168.1.110&leased_mac_address=AA:BB:CC:DD:EE:F2&expired=0&remote_id=test9"
	getTestHandler[10].user = "test"
	getTestHandler[10].pass = "test"
	getTestHandler[10].status = http.StatusBadRequest
	getTestHandler[10].routerIP = "5.5.5"

	for k := range getTestHandler {

		logger.Println("")
		logger.Println("")
		logger.Println("")

		x := httptest.NewRequest("GET", getTestHandler[k].URI, nil)
		x.SetBasicAuth(getTestHandler[k].user, getTestHandler[k].pass)
		rr := httptest.NewRecorder()

		if k == 10 {
			x.RemoteAddr = getTestHandler[k].routerIP
		}

	//	handler := http.HandlerFunc(BatchModeEndpointRouter)
		handler.ServeHTTP(rr, x)

		if status := rr.Code; status != getTestHandler[k].status {
			t.Errorf("%v - get handler returned wrong status code: got %v want %v", k, status, getTestHandler[k].status)
			t.Errorf("uri: %v", getTestHandler[k].URI)
		}
	}
	logger.Println("")
	logger.Println("")
	logger.Println("")

	// post requests

	var postTestHandler [13]mockPostRequest

	// good requests
	postTestHandler[0].Expired = "0"
	postTestHandler[0].IPAddress = "192.168.1.10"
	postTestHandler[0].RemoteID = "test1"
	postTestHandler[0].LeasedMacAddress = "AA:BB:CC:DD:EE:E0"
	postTestHandler[0].routerIP = mockRouter
	postTestHandler[0].status = http.StatusOK
	postTestHandler[0].user = "test"
	postTestHandler[0].pass = "test"

	postTestHandler[1].Expired = "1"
	postTestHandler[1].IPAddress = "192.168.1.20"
	postTestHandler[1].RemoteID = "test2"
	postTestHandler[1].LeasedMacAddress = "AA:BB:CC:DD:EE:E1"
	postTestHandler[1].routerIP = mockRouter
	postTestHandler[1].status = http.StatusOK
	postTestHandler[1].user = "test"
	postTestHandler[1].pass = "test"

	postTestHandler[2].Expired = "0"
	postTestHandler[2].IPAddress = "192.168.1.30"
	postTestHandler[2].RemoteID = ""
	postTestHandler[2].LeasedMacAddress = "AA:BB:CC:DD:EE:E2"
	postTestHandler[2].routerIP = mockRouter
	postTestHandler[2].status = http.StatusOK
	postTestHandler[2].user = "test"
	postTestHandler[2].pass = "test"

	postTestHandler[3].Expired = "1"
	postTestHandler[3].IPAddress = "192.168.1.40"
	postTestHandler[3].RemoteID = ""
	postTestHandler[3].LeasedMacAddress = "AA:BB:CC:DD:EE:E3"
	postTestHandler[3].routerIP = mockRouter
	postTestHandler[3].status = http.StatusOK
	postTestHandler[3].user = "test"
	postTestHandler[3].pass = "test"

	// bad requests

	// malformed IP address
	postTestHandler[4].Expired = "1"
	postTestHandler[4].IPAddress = "192.168.1"
	postTestHandler[4].RemoteID = ""
	postTestHandler[4].LeasedMacAddress = "AA:BB:CC:DD:EE:E4"
	postTestHandler[4].routerIP = mockRouter
	postTestHandler[4].status = http.StatusBadRequest
	postTestHandler[4].user = "test"
	postTestHandler[4].pass = "test"

	// malformed MAC address
	postTestHandler[5].Expired = "1"
	postTestHandler[5].IPAddress = "192.168.1.60"
	postTestHandler[5].RemoteID = ""
	postTestHandler[5].LeasedMacAddress = "AA:BB:CC:DD:EE"
	postTestHandler[5].routerIP = mockRouter
	postTestHandler[5].status = http.StatusBadRequest
	postTestHandler[5].user = "test"
	postTestHandler[5].pass = "test"

	// missing expired
	postTestHandler[6].Expired = ""
	postTestHandler[6].IPAddress = "192.168.1.70"
	postTestHandler[6].RemoteID = ""
	postTestHandler[6].LeasedMacAddress = "AA:BB:CC:DD:EE:E6"
	postTestHandler[6].routerIP = mockRouter
	postTestHandler[6].status = http.StatusBadRequest
	postTestHandler[6].user = "test"
	postTestHandler[6].pass = "test"

	// non-int expired
	postTestHandler[7].Expired = "test"
	postTestHandler[7].IPAddress = "192.168.1.80"
	postTestHandler[7].RemoteID = ""
	postTestHandler[7].LeasedMacAddress = "AA:BB:CC:DD:EE:E7"
	postTestHandler[7].routerIP = mockRouter
	postTestHandler[7].status = http.StatusBadRequest
	postTestHandler[7].user = "test"
	postTestHandler[7].pass = "test"

	// expired not between 0 and 1
	postTestHandler[8].Expired = "2"
	postTestHandler[8].IPAddress = "192.168.1.90"
	postTestHandler[8].RemoteID = ""
	postTestHandler[8].LeasedMacAddress = "AA:BB:CC:DD:EE:E8"
	postTestHandler[8].routerIP = mockRouter
	postTestHandler[8].status = http.StatusBadRequest
	postTestHandler[8].user = "test"
	postTestHandler[8].pass = "test"

	// remoteID is too long, when trying to expire an entry
	postTestHandler[9].Expired = "0"
	postTestHandler[9].IPAddress = "192.168.1.100"
	postTestHandler[9].RemoteID = longStringOne
	postTestHandler[9].LeasedMacAddress = "AA:BB:CC:DD:EE:E9"
	postTestHandler[9].routerIP = mockRouter
	postTestHandler[9].status = http.StatusBadRequest
	postTestHandler[9].user = "test"
	postTestHandler[9].pass = "test"

	// remoteID is too long, when trying to add a new entry
	postTestHandler[10].Expired = "1"
	postTestHandler[10].IPAddress = "192.168.1.110"
	postTestHandler[10].RemoteID = longStringTwo
	postTestHandler[10].LeasedMacAddress = "AA:BB:CC:DD:EE:EA"
	postTestHandler[10].routerIP = mockRouter
	postTestHandler[10].status = http.StatusBadRequest
	postTestHandler[10].user = "test"
	postTestHandler[10].pass = "test"

	// auth failure (user name and password mismatch)
	postTestHandler[11].Expired = "0"
	postTestHandler[11].IPAddress = "192.168.1.120"
	postTestHandler[11].RemoteID = ""
	postTestHandler[11].LeasedMacAddress = "AA:BB:CC:DD:EE:EB"
	postTestHandler[11].routerIP = mockRouter
	postTestHandler[11].status = http.StatusUnauthorized
	postTestHandler[11].user = "fail"
	postTestHandler[11].pass = "fail"

	// unable to parse router IP.
	postTestHandler[12].Expired = "0"
	postTestHandler[12].IPAddress = "192.168.1.130"
	postTestHandler[12].RemoteID = ""
	postTestHandler[12].LeasedMacAddress = "AA:BB:CC:DD:EE:EC"
	postTestHandler[12].routerIP = "6.6.6"
	postTestHandler[12].status = http.StatusBadRequest
	postTestHandler[12].user = "test"
	postTestHandler[12].pass = "test"

	for k, _ := range postTestHandler {

		logger.Println("")
		logger.Println("")
		logger.Println("")

		payload, err := json.Marshal(postTestHandler[k])

		if err != nil {
			t.Errorf("error marshalling JSON to payload : %v",err.Error())
		}

		x := httptest.NewRequest("post", "/api/dhcp_assignments", bytes.NewBuffer(payload))
		x.Header.Set("Content-Type", "application/json")
		x.SetBasicAuth(postTestHandler[k].user, postTestHandler[k].pass)
		rr := httptest.NewRecorder()

		if k == 12 {
			x.RemoteAddr = postTestHandler[k].routerIP
		}

		handler.ServeHTTP(rr, x)

		if status := rr.Code; status != postTestHandler[k].status {
			t.Errorf("%v - post handler returned wrong status code: got %v want %v", k, status, postTestHandler[k].status)
		}
	}

}
