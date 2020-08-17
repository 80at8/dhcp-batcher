package main

import (
	dhcp "github.com/krolaw/dhcp4"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//
// in memory lease information, lease times are automatically trimmed until they hit zero, then they're flagged as
// expired.
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
	isExpired string
}

type leaseRecord struct {
	entry map[string]lease
	mutex sync.RWMutex
}

func (l *leaseRecord) addLease(MAC, IP string, leaseTime uint32, options dhcp.Options) {

	var r net.IP = options[dhcp.OptionRouter]
	a := lease{
		mac:       MAC,
		ip:        IP,
		router:    r.To4(),
		leaseTime: leaseTime,
		timeStamp: time.Now(),
		isExpired: "0",
	}

	if opt82len := len(options[dhcp.OptionRelayAgentInformation]); opt82len > 2 {
		o := options[dhcp.OptionRelayAgentInformation]
		s1 := o[1]
		a.cid = string(o[2:2+s1])
		a.rid = string(o[len(a.cid)+4:opt82len])

	}

	l.mutex.Lock()
	l.entry[MAC] = a
    l.mutex.Unlock()

    if logger.GetLevel() == logrus.DebugLevel {
		logger.Debug("renewal lease time is : ", options[dhcp.OptionRenewalTimeValue])
		l.printDebug()
	} else {
		l.printInfo()
	}

}

func (l *leaseRecord) init() {
	l.entry = make(map[string]lease)
}


// flag expired leasetimes for batching to Sonar
func (l *leaseRecord) trim(ctl chan bool) {
	// define a 10 second trim timer
	logger.Info("lease trim started")

	t := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctl:
			logger.Info("lease trim: exit..")
			return
		case <-t.C:
			updated, expired := 0,0
			l.mutex.Lock()
			for k,v := range l.entry {
				if v.leaseTime < 10 && v.isExpired == "0"{
					v.leaseTime = 0
					v.isExpired = "1"
					l.entry[k] = v
					expired++
				} else {
					if v.isExpired != "1" {
						v.isExpired = "0"
						v.leaseTime -= 10
						l.entry[k] = v
						updated++
					}
				}
			}
			l.mutex.Unlock()
			if logger.GetLevel() == logrus.DebugLevel {
				l.printDebug()
				logger.Debug("trim - ", updated, " updated, ", expired, " expired")
			} else {
				l.printInfo()
				logger.Info("trim - ", updated, " updated, ", expired, " expired")
			}
		}
	}
}

func (l *leaseRecord) printDebug() {
	for i,v := range l.entry {
		logger.Debug("")
		logger.Debug("    leaseRecord: ", i)
		logger.Debug("     circuit id: ", v.cid)
		logger.Debug("      remote id: ", v.rid)
		logger.Debug("      timestamp: ", v.timeStamp)
		logger.Debug("     lease time: ", v.leaseTime)
		logger.Debug("             IP: ", v.ip)
		logger.Debug("      router IP: ", v.router.String())
		logger.Debug("            MAC: ", v.mac)
		logger.Debug("     is expired: ", v.isExpired)
		logger.Debug("")
	}
}

func (l *leaseRecord) printInfo() {
	for _,v := range l.entry {
		logger.Info("IP: ", v.ip, " MAC: ", v.mac, " expired: ", v.isExpired)
	}
}