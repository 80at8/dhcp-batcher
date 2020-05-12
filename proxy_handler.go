package main

import (
	"context"
	dhcp "github.com/krolaw/dhcp4"
	"net"
	"os"
	"os/signal"
	"time"
)

var dhcpServers []net.IP
var dhcpGIAddr net.IP

type DHCPHandler struct {
	m map[string]bool
}

func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	switch msgType {

	case dhcp.Discover:
		logger.Info("discover ", p.YIAddr(), "from", p.CHAddr())
		h.m[string(p.XId())] = true
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetGIAddr(dhcpGIAddr)
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
		logger.Info("offering from", sip.String(), p.YIAddr(), "to", p.CHAddr())
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
		logger.Info("request ", p.YIAddr(), "from", p.CHAddr())
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetFile(p.File())
		p2.SetCIAddr(p.CIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetGIAddr(dhcpGIAddr)
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
		logger.Info("ACK from", sip.String(), p.YIAddr(), "to", p.CHAddr())
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
		return p2

	case dhcp.NAK:
		if !h.m[string(p.XId())] {
			return nil
		}
		logger.Info("NAK from", p.SIAddr(), p.YIAddr(), "to", p.CHAddr())
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
		return p2

	case dhcp.Release, dhcp.Decline:
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetFile(p.File())
		p2.SetCIAddr(p.CIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetGIAddr(dhcpGIAddr)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2
	}
	return nil
}

func createRelay(in, out string, ctl chan bool) {
	handler := &DHCPHandler{m: make(map[string]bool)}
	go ListenAndServeIf(in, out, 67, handler)
	go ListenAndServeIf(out, in, 68, handler)

	// listen for stop signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	// true, exit batchScheduler
	ctl <- true

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("proxy: exit..")
	return

}
