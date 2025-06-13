//go:build rp2350

//----------------------------------------------------------------------
// This file is part of srv9p.
// Copyright (C) 2024-present Bernd Fix   >Y<
//
// srv9p is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// srv9p is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL3.0-or-later
//----------------------------------------------------------------------

package srv9p

import (
	"errors"
	"io"
	"log/slog"
	"machine"
	"net"
	"net/netip"
	"time"

	"github.com/soypat/cyw43439"
	"github.com/soypat/seqs/eth/dhcp"
	"github.com/soypat/seqs/eth/dns"
	"github.com/soypat/seqs/stacks"
)

// Raspberry Pico2 W  [RP2350]
type Pico2WDevice struct {
	ref *cyw43439.Device // reference to device
}

// LED on or off (if applicable)
func (dev *Pico2WDevice) LED(on bool) {
	dev.ref.GPIOSet(0, on)
}

// Initialize device
func InitDevice() Device {
	// access device
	dev := new(Pico2WDevice)
	dev.ref = cyw43439.NewPicoWDevice()
	return dev
}

// SetupListener returns a TCP listener on the given port.
// Can connect to WiFi hotspot (if applicable) first.
// If DHCP fails, a static IP can be used.
func SetupListener(dev Device, host, ip, ssid, passwd string, port uint16) (lst net.Listener, state int) {
	d, ok := dev.(*Pico2WDevice)
	if !ok {
		state = StatDEV
		return
	}

	var logger *slog.Logger = slog.New(slog.NewTextHandler(machine.Serial, &slog.HandlerOptions{Level: slog.LevelDebug - 1}))
	time.Sleep(2 * time.Second)

	var stack *stacks.PortStack
	if stack, state = SetupWithDHCP(d.ref, SetupConfig{
		Hostname:    host,
		RequestedIP: ip,
		TCPPorts:    1,
		SSID:        ssid,
		Passwd:      passwd,
		Logger:      logger,
	}); state != StatOK {
		return
	}
	listener, err := stacks.NewTCPListener(stack, stacks.TCPListenerConfig{
		MaxConnections: 3,
		ConnTxBufSize:  512,
		ConnRxBufSize:  512,
	})
	if err != nil {
		state = StatLISTEN1
		return
	}
	lst = listener
	if listener.StartListening(port) != nil {
		state = StatLISTEN2
	}
	return
}

//======================================================================
// copied from https://raw.githubusercontent.com/soypat/cyw43439,
// file '/examples/common/common.go'.
//======================================================================

const mtu = cyw43439.MTU

type SetupConfig struct {
	// DHCP requested hostname.
	Hostname string
	// DHCP requested IP address. On failing to find DHCP server is used as static IP.
	RequestedIP string
	Logger      *slog.Logger
	// Number of UDP ports to open for the stack. (we'll actually open one more than this for DHCP)
	UDPPorts uint16
	// Number of TCP ports to open for the stack.
	TCPPorts uint16

	SSID   string
	Passwd string
}

func SetupWithDHCP(dev *cyw43439.Device, cfg SetupConfig) (*stacks.PortStack, int) {
	cfg.UDPPorts++ // Add extra UDP port for DHCP client.
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.Level(127), // Make temporary logger that does no logging.
		}))
	}
	var err error
	var reqAddr netip.Addr
	if cfg.RequestedIP != "" {
		reqAddr, err = netip.ParseAddr(cfg.RequestedIP)
		if err != nil {
			return nil, StatIP
		}
	}

	wificfg := cyw43439.DefaultWifiConfig()
	wificfg.Logger = logger
	// cfg.Logger = logger // Uncomment to see in depth info on wifi device functioning.
	logger.Info("initializing pico W device...")
	devInitTime := time.Now()

	if err = dev.Init(wificfg); err != nil {
		return nil, StatWIFI
	}
	logger.Info("cyw43439:Init", slog.Duration("duration", time.Since(devInitTime)))
	if len(cfg.Passwd) == 0 {
		logger.Info("joining open network:", slog.String("ssid", cfg.SSID))
	} else {
		logger.Info("joining WPA secure network", slog.String("ssid", cfg.SSID), slog.Int("passlen", len(cfg.Passwd)))
	}
	for range 5 {
		// Set ssid/pass in secrets.go
		err = dev.JoinWPA2(cfg.SSID, cfg.Passwd)
		if err == nil {
			break
		}
		logger.Error("wifi join failed", slog.String("err", err.Error()))
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		return nil, StatWPA2
	}
	mac, _ := dev.HardwareAddr6()
	logger.Info("wifi join success!", slog.String("mac", net.HardwareAddr(mac[:]).String()))

	stack := stacks.NewPortStack(stacks.PortStackConfig{
		MAC:             mac,
		MaxOpenPortsUDP: int(cfg.UDPPorts),
		MaxOpenPortsTCP: int(cfg.TCPPorts),
		MTU:             mtu,
		Logger:          logger,
	})

	dev.RecvEthHandle(stack.RecvEth)

	// Begin asynchronous packet handling.
	go nicLoop(dev, stack)

	// Perform DHCP request.
	dhcpClient := stacks.NewDHCPClient(stack, dhcp.DefaultClientPort)
	err = dhcpClient.BeginRequest(stacks.DHCPRequestConfig{
		RequestedAddr: reqAddr,
		Xid:           uint32(time.Now().Nanosecond()),
		Hostname:      cfg.Hostname,
	})
	if err != nil {
		return stack, StatDHCP1
	}
	i := 0
	for dhcpClient.State() != dhcp.StateBound {
		i++
		logger.Info("DHCP ongoing...")
		time.Sleep(time.Second / 2)
		if i > 15 {
			if !reqAddr.IsValid() {
				return stack, StatDHCP2
			}
			logger.Info("DHCP did not complete, assigning static IP", slog.String("ip", cfg.RequestedIP))
			stack.SetAddr(reqAddr)
			return stack, StatOK
		}
	}
	var primaryDNS netip.Addr
	dnsServers := dhcpClient.DNSServers()
	if len(dnsServers) > 0 {
		primaryDNS = dnsServers[0]
	}
	ip := dhcpClient.Offer()
	logger.Info("DHCP complete",
		slog.Uint64("cidrbits", uint64(dhcpClient.CIDRBits())),
		slog.String("ourIP", ip.String()),
		slog.String("dns", primaryDNS.String()),
		slog.String("broadcast", dhcpClient.BroadcastAddr().String()),
		slog.String("gateway", dhcpClient.Gateway().String()),
		slog.String("router", dhcpClient.Router().String()),
		slog.String("dhcp", dhcpClient.DHCPServer().String()),
		slog.String("hostname", string(dhcpClient.Hostname())),
		slog.Duration("lease", dhcpClient.IPLeaseTime()),
		slog.Duration("renewal", dhcpClient.RenewalTime()),
		slog.Duration("rebinding", dhcpClient.RebindingTime()),
	)

	stack.SetAddr(ip) // It's important to set the IP address after DHCP completes.
	return stack, StatOK
}

// ResolveHardwareAddr obtains the hardware address of the given IP address.
func ResolveHardwareAddr(stack *stacks.PortStack, ip netip.Addr) ([6]byte, error) {
	if !ip.IsValid() {
		return [6]byte{}, errors.New("invalid ip")
	}
	arpc := stack.ARP()
	arpc.Abort() // Remove any previous ARP requests.
	err := arpc.BeginResolve(ip)
	if err != nil {
		return [6]byte{}, err
	}
	time.Sleep(4 * time.Millisecond)
	// ARP exchanges should be fast, don't wait too long for them.
	const timeout = time.Second
	const maxretries = 20
	retries := maxretries
	for !arpc.IsDone() && retries > 0 {
		retries--
		if retries == 0 {
			return [6]byte{}, errors.New("arp timed out")
		}
		time.Sleep(timeout / maxretries)
	}
	_, hw, err := arpc.ResultAs6()
	return hw, err
}

type Resolver struct {
	stack     *stacks.PortStack
	dns       *stacks.DNSClient
	dhcp      *stacks.DHCPClient
	dnsaddr   netip.Addr
	dnshwaddr [6]byte
}

func NewResolver(stack *stacks.PortStack, dhcp *stacks.DHCPClient) (*Resolver, error) {
	dnsc := stacks.NewDNSClient(stack, dns.ClientPort)
	dnsaddrs := dhcp.DNSServers()
	if len(dnsaddrs) > 0 && !dnsaddrs[0].IsValid() {
		return nil, errors.New("dns addr obtained via DHCP not valid")
	}
	return &Resolver{
		stack:   stack,
		dhcp:    dhcp,
		dns:     dnsc,
		dnsaddr: dnsaddrs[0],
	}, nil
}

func (r *Resolver) LookupNetIP(host string) ([]netip.Addr, error) {
	name, err := dns.NewName(host)
	if err != nil {
		return nil, err
	}
	err = r.updateDNSHWAddr()
	if err != nil {
		return nil, err
	}

	err = r.dns.StartResolve(r.dnsConfig(name))
	if err != nil {
		return nil, err
	}
	time.Sleep(5 * time.Millisecond)
	retries := 100

	for retries > 0 {
		done, _ := r.dns.IsDone()
		if done {
			break
		}
		retries--
		time.Sleep(20 * time.Millisecond)
	}
	done, rcode := r.dns.IsDone()
	if !done && retries == 0 {
		return nil, errors.New("dns lookup timed out")
	} else if rcode != dns.RCodeSuccess {
		return nil, errors.New("dns lookup failed:" + rcode.String())
	}
	answers := r.dns.Answers()
	if len(answers) == 0 {
		return nil, errors.New("no dns answers")
	}
	var addrs []netip.Addr
	for i := range answers {
		data := answers[i].RawData()
		if len(data) == 4 {
			addrs = append(addrs, netip.AddrFrom4([4]byte(data)))
		}
	}
	if len(addrs) == 0 {
		return nil, errors.New("no ipv4 dns answers")
	}
	return addrs, nil
}

func (r *Resolver) updateDNSHWAddr() (err error) {
	r.dnshwaddr, err = ResolveHardwareAddr(r.stack, r.dnsaddr)
	return err
}

func (r *Resolver) dnsConfig(name dns.Name) stacks.DNSResolveConfig {
	return stacks.DNSResolveConfig{
		Questions: []dns.Question{
			{
				Name:  name,
				Type:  dns.TypeA,
				Class: dns.ClassINET,
			},
		},
		DNSAddr:         r.dnsaddr,
		DNSHWAddr:       r.dnshwaddr,
		EnableRecursion: true,
	}
}

func nicLoop(dev *cyw43439.Device, Stack *stacks.PortStack) {
	// Maximum number of packets to queue before sending them.
	const (
		queueSize                = 3
		maxRetriesBeforeDropping = 3
	)
	var queue [queueSize][mtu]byte
	var lenBuf [queueSize]int
	var retries [queueSize]int
	markSent := func(i int) {
		queue[i] = [mtu]byte{} // Not really necessary.
		lenBuf[i] = 0
		retries[i] = 0
	}
	for {
		stallRx := true
		// Poll for incoming packets.
		for i := 0; i < 1; i++ {
			gotPacket, err := dev.PollOne()
			if err != nil {
				println("poll error:", err.Error())
			}
			if !gotPacket {
				break
			}
			stallRx = false
		}

		// Queue packets to be sent.
		for i := range queue {
			if retries[i] != 0 {
				continue // Packet currently queued for retransmission.
			}
			var err error
			buf := queue[i][:]
			lenBuf[i], err = Stack.HandleEth(buf[:])
			if err != nil {
				println("stack error n(should be 0)=", lenBuf[i], "err=", err.Error())
				lenBuf[i] = 0
				continue
			}
			if lenBuf[i] == 0 {
				break
			}
		}
		stallTx := lenBuf == [queueSize]int{}
		if stallTx {
			if stallRx {
				// Avoid busy waiting when both Rx and Tx stall.
				time.Sleep(51 * time.Millisecond)
			}
			continue
		}

		// Send queued packets.
		for i := range queue {
			n := lenBuf[i]
			if n <= 0 {
				continue
			}
			err := dev.SendEth(queue[i][:n])
			if err != nil {
				// Queue packet for retransmission.
				retries[i]++
				if retries[i] > maxRetriesBeforeDropping {
					markSent(i)
					println("dropped outgoing packet:", err.Error())
				}
			} else {
				markSent(i)
			}
		}
	}
}
