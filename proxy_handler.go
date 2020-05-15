package main

import (
	"context"
	dhcp "github.com/krolaw/dhcp4"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"
)

var dhcpServers []net.IP
var proxyServerIP net.IP
var downStreamGIAddr net.IP

type DHCPHandler struct {
	m map[string]bool
}

func proxyGIAddrLookup() {

}


func addProxyGIAddr() {

}



func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	switch msgType {
		//CIADDR (Client IP address)
		//YIADDR (Your IP address)
		//SIADDR (Server IP address)
		//GIADDR (Gateway IP address)
		//CHADDR (Client hardware address)
	case dhcp.Discover:
		logger.Info("DISCOVER ", p.YIAddr(), " from ", p.CHAddr())
		logger.Debug("giaddr is  ",p.GIAddr())
		logger.Debug("flags are " , p.Flags())
		downStreamGIAddr = p.GIAddr()
		h.m[string(p.XId())] = true
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetGIAddr(proxyServerIP)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.Offer:
		if !h.m[string(p.XId())] {
			return nil
		}
		var sip net.IP
		for k, v := range p.ParseOptions() {
			if k == dhcp.OptionServerIdentifier {
				sip = v
			}
		}
		logger.Info("OFFER from ", sip.String()," ", p.YIAddr(), " to ", p.CHAddr())
		logger.Debug("giaddr is  ",p.GIAddr())
		logger.Debug("flags are " , p.Flags())
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetYIAddr(p.YIAddr())
	    p2.SetGIAddr(p.GIAddr())
	    p2.SetSIAddr(p.SIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.Request:
		h.m[string(p.XId())] = true
		logger.Info("REQUEST ", p.YIAddr(), " from ", p.CHAddr())
		logger.Debug("giaddr is  ",proxyServerIP)
		logger.Debug("flags are " , p.Flags())
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetFile(p.File())
		p2.SetCIAddr(p.CIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetGIAddr(proxyServerIP)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.ACK:
		if !h.m[string(p.XId())] {
			return nil
		}
		var sip net.IP
		for k, v := range p.ParseOptions() {
			if k == dhcp.OptionServerIdentifier {
				sip = v
			}
		}
		logger.Info("ACK from ", sip.String()," ",p.YIAddr(), " to ", p.CHAddr())
		logger.Debug("giaddr is  ",p.GIAddr())
		logger.Debug("flags are " , p.Flags())
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetSIAddr(p.SIAddr())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}

		go batchTable.UpdateBatchTable("0", downStreamGIAddr, p.CHAddr(), p.YIAddr(), "")  // active
		return p2

	case dhcp.NAK:
		if !h.m[string(p.XId())] {
			return nil
		}
		logger.Info("NAK from ", p.SIAddr()," ", p.YIAddr(), " to ", p.CHAddr())
		logger.Debug("giaddr is  ",p.GIAddr())
		logger.Debug("flags are " , p.Flags())
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetSIAddr(p.SIAddr())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		go batchTable.UpdateBatchTable("1", downStreamGIAddr, p.CHAddr(), p.YIAddr(),  "") // expired
		return p2

	case dhcp.Release, dhcp.Decline:
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetFile(p.File())
		p2.SetCIAddr(p.CIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetGIAddr(proxyServerIP)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		go batchTable.UpdateBatchTable("1", downStreamGIAddr, p.CHAddr(), p.YIAddr(),  "") // expired
		return p2
	}
	return nil
}

func startDHCPProxy(ctl chan bool) {
	servers := strings.Fields(*batchProxyOptions.upstreamServerIPs)
	for _, s := range servers {
		dhcpServers = append(dhcpServers, net.ParseIP(s))
	}
	proxyServerIP = net.ParseIP(*batchProxyOptions.proxyServerIP)
	handler := &DHCPHandler{m: make(map[string]bool)}

	if batchProxyOptions.isProxySingle {
		go ListenAndServeIf(*batchProxyOptions.proxySingleInterface, *batchProxyOptions.proxySingleInterface, 67, handler)
		go ListenAndServeIf(*batchProxyOptions.proxySingleInterface, *batchProxyOptions.proxySingleInterface, 68, handler)
	} else {
		go ListenAndServeIf(*batchProxyOptions.proxyUpstreamInterface, *batchProxyOptions.proxyDownstreamInterface, 67, handler)
		go ListenAndServeIf(*batchProxyOptions.proxyDownstreamInterface, *batchProxyOptions.proxyUpstreamInterface, 68, handler)
	}

	// listen for stop signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	// true, exit batchScheduler
	ctl <- true

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Println()
	logger.Info("proxy: exit..")
	return

}
