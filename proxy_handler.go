package main

import (
	"context"
	"encoding/binary"
	dhcp "github.com/krolaw/dhcp4"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"
)

//2.0 Relay Agent Information Option
//
//This document defines a new DHCP Option called the Relay Agent
//Information Option.  It is a "container" option for specific agent-
//supplied sub-options.  The format of the Relay Agent Information
//option is:
//
//Code   Len     Agent Information Field
//+------+------+------+------+------+------+--...-+------+
//|  82  |   N  |  i1  |  i2  |  i3  |  i4  |      |  iN  |
//+------+------+------+------+------+------+--...-+------+
//
//The length N gives the total number of octets in the Agent
//Information Field.  The Agent Information field consists of a
//sequence of SubOpt/Length/Value tuples for each sub-option, encoded
//in the following manner:
//
//SubOpt  Len     Sub-option Value
//+------+------+------+------+------+------+--...-+------+
//|  1   |   N  |  s1  |  s2  |  s3  |  s4  |      |  sN  |
//+------+------+------+------+------+------+--...-+------+
//SubOpt  Len     Sub-option Value
//+------+------+------+------+------+------+--...-+------+
//|  2   |   N  |  i1  |  i2  |  i3  |  i4  |      |  iN  |
//+------+------+------+------+------+------+--...-+------+
//
//No "pad" sub-option is defined, and the Information field shall NOT
//be terminated with a 255 sub-option.  The length N of the DHCP Agent
//Information Option shall include all bytes of the sub-option
//code/length/value tuples.  Since at least one sub-option must be
//defined, the minimum Relay Agent Information length is two (2).  The
//length N of the sub-options shall be the number of octets in only
//that sub-option's value field.  A sub-option length may be zero.  The
//sub-options need not appear in sub-option code order.
//
//The initial assignment of DHCP Relay Agent Sub-options is as follows:
//
//DHCP Agent              Sub-Option Description
//Sub-option Code
//---------------         ----------------------
//1                   Agent Circuit ID Sub-option
//2                   Agent Remote ID Sub-option
//
//
//
//

//Patrick                     Standards Track                     [Page 5]
//
//RFC 3046          DHCP Relay Agent Information Option       January 2001
//
//
//2.1 Agent Operation
//
//Overall adding of the DHCP relay agent option SHOULD be configurable,
//and SHOULD be disabled by default.  Relay agents SHOULD have separate
//configurables for each sub-option to control whether it is added to
//client-to-server packets.
//
//A DHCP relay agent adding a Relay Agent Information field SHALL add
//it as the last option (but before 'End Option' 255, if present) in
//the DHCP options field of any recognized BOOTP or DHCP packet
//forwarded from a client to a server.
//
//Relay agents receiving a DHCP packet from an untrusted circuit with
//giaddr set to zero (indicating that they are the first-hop router)
//but with a Relay Agent Information option already present in the
//packet SHALL discard the packet and increment an error count.  A
//trusted circuit may contain a trusted downstream (closer to client)
//network element (bridge) between the relay agent and the client that
//MAY add a relay agent option but not set the giaddr field.  In this
//case, the relay agent does NOT add a "second" relay agent option, but
//forwards the DHCP packet per normal DHCP relay agent operations,
//setting the giaddr field as it deems appropriate.
//
//The mechanisms for distinguishing between "trusted" and "untrusted"
//circuits are specific to the type of circuit termination equipment,
//and may involve local administration.  For example, a Cable Modem
//Termination System may consider upstream packets from most cable
//modems as "untrusted", but an ATM switch terminating VCs switched
//through a DSLAM may consider such VCs as "trusted" and accept a relay
//agent option added by the DSLAM.
//
//Relay agents MAY have a configurable for the maximum size of the DHCP
//packet to be created after appending the Agent Information option.
//Packets which, after appending the Relay Agent Information option,
//would exceed this configured maximum size shall be forwarded WITHOUT
//adding the Agent Information option.  An error counter SHOULD be
//incremented in this case.  In the absence of this configurable, the
//agent SHALL NOT increase a forwarded DHCP packet size to exceed the
//MTU of the interface on which it is forwarded.
//
//The Relay Agent Information option echoed by a server MUST be removed
//by either the relay agent or the trusted downstream network element
//which added it when forwarding a server-to-client response back to
//the client.
//
//
//

var dhcpServers []net.IP
var proxyServerIP net.IP

type DHCPHandler struct {
	m map[string]bool
}


func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {

	packetOptions := p.ParseOptions()

	switch msgType {
	//CIADDR (Client IP address)
	//YIADDR (Your IP address)
	//SIADDR (Server IP address)
	//GIADDR (Gateway IP address)
	//CHADDR (Client hardware address)
	case dhcp.Discover:
		logger.Debug("DISCOVER ", p.YIAddr(), " from ", p.CHAddr())
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
		logger.Debug("OFFER")
		if !h.m[string(p.XId())] {
			return nil
		}
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())

		for k, v := range packetOptions {
			//if k == dhcp.OptionClientIdentifier || k == dhcp.OptionHostName {
			//	logger.Debug("Option: ", k, " is ", string(v))
			//} else {
			//	logger.Debug("Option: ", k, " is ", v)
			//}
			p2.AddOption(k, v)
		}
		p2.AddOption(dhcp.OptionServerIdentifier, []byte(proxyServerIP.To4()))

		return p2

	case dhcp.Request:
		h.m[string(p.XId())] = true
		logger.Info("REQUEST ", p.YIAddr(), " from ", p.CHAddr())
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetFile(p.File())
		p2.SetCIAddr(p.CIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetGIAddr(proxyServerIP)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)

		for k, v := range packetOptions {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.ACK:
		if !h.m[string(p.XId())] {
			return nil
		}
		logger.Debug("ACK")
		leaseTable.addLease(p.CHAddr().String(), p.YIAddr().String(),binary.BigEndian.Uint32(options[dhcp.OptionIPAddressLeaseTime]), packetOptions)
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetSIAddr(p.SIAddr())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range packetOptions {
			p2.AddOption(k, v)
		}
		p2.AddOption(dhcp.OptionServerIdentifier, []byte(proxyServerIP.To4()))
		//batchTable.UpdateBatchTable("0", downStreamGIAddr, p.CHAddr(), p.YIAddr(), "") // active
		return p2

	case dhcp.NAK:
		if !h.m[string(p.XId())] {
			return nil
		}
		logger.Info("NAK from ", p.SIAddr(), " ", p.YIAddr(), " to ", p.CHAddr())
		logger.Debug("giaddr is  ", p.GIAddr())
		logger.Debug("flags are ", p.Flags())
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetSIAddr(p.SIAddr())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range packetOptions {
			if k == dhcp.OptionServerIdentifier {
				p2.AddOption(k, []byte(proxyServerIP.To4()))
			} else {
				p2.AddOption(k, v)
			}
		}
		//		go batchTable.UpdateBatchTable("1", downStreamGIAddr, p.CHAddr(), p.YIAddr(),  "") // expired
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
		for k, v := range packetOptions {
			p2.AddOption(k, v)
		}
		//	go batchTable.UpdateBatchTable("1", downStreamGIAddr, p.CHAddr(), p.YIAddr(),  "") // expired
		return p2
	}
	return nil
}

func startDHCPProxy(ctl chan bool) {
	leaseTable.init()
	go leaseTable.trim(ctl)
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
