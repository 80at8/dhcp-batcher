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
	MAC net.HardwareAddr
	IP net.IP
	Router net.IP
	time uint32
	timestamp uint32
	packet dhcp.Packet
}

type leaseRecord struct {
	entry map[string]lease
	mutex sync.RWMutex
}

func (l *leaseRecord) addLease(MAC net.HardwareAddr, IP, Router net.IP, leaseTime []byte) {
	l.mutex.Lock()
	l.entry[string(MAC)] = lease {
		time:binary.BigEndian.Uint32(leaseTime),
		IP: IP,
		Router: Router,
		timestamp: uint32(time.Now().Unix()),
	}
	l.mutex.Unlock()
}

func (l *leaseRecord) init() {
		l.entry = make(map[string]lease)
}

