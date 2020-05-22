package main

import (
	"encoding/binary"
	dhcp "github.com/krolaw/dhcp4"
	"net"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// holds lease times
//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var leaseTable leaseRecord

type lease struct {
	mac string
	ip string
	router net.IP
	cid string
	rid string
	leaseTime uint32
	timeStamp time.Time
}

type leaseRecord struct {
	entry map[string]lease
	mutex sync.RWMutex
}

func (l *leaseRecord) addLease(MAC, IP string, options dhcp.Options) {
	var r net.IP = options[dhcp.OptionRouter]
	a := lease{
		mac:       MAC,
		ip:        IP,
		router:    r.To4(),
		leaseTime: binary.BigEndian.Uint32(options[dhcp.OptionIPAddressLeaseTime]),
		timeStamp: time.Now(),
	}

	if opt82len := len(options[dhcp.OptionRelayAgentInformation]); opt82len > 2 {
		o := options[dhcp.OptionRelayAgentInformation]
		// TODO should probably put some bounds checking in here
		s1 := o[1]
		a.cid = string(o[2:2+s1])
		s2 := o[3+s1]
		a.rid = string(o[len(o)-int(s2):])
	}

	l.mutex.Lock()
	l.entry[MAC] = a
    l.mutex.Unlock()

	for i,v := range l.entry {
		logger.Debug("    ************")
		logger.Debug("    leaseRecord: ", i)
		logger.Debug("     circuit id: ", v.cid)
		logger.Debug("      remote id: ", v.rid)
		logger.Debug("      timestamp: ", v.timeStamp)
		logger.Debug("     lease time: ", v.leaseTime)
		logger.Debug("             IP: ", v.ip)
		logger.Debug("      router IP: ", v.router.String())
		logger.Debug("            MAC: ", v.mac)
		logger.Debug("    ************")
	}

}

func (l *leaseRecord) init() {
	l.entry = make(map[string]lease)
}


// flag expired leasetimes for batching to Sonar
func (l *leaseRecord) trim() {
	// define a 10 second trim timer
}