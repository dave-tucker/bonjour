package bonjour

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/socketplane/go-fastping"
)

type response struct {
	addr *net.IPAddr
	rtt  time.Duration
}

const (
	// EchoReply indicates that a reply was received
	EchoReply = iota
	// NoReply indicates that no reply was received
	NoReply
	// Error indicates that there was an error with the request
	Error
)

func echo(address string, ip *net.IP) (int, error) {
	p := fastping.NewPinger()
	p.Debug = false
	netProto := "ip4:icmp"
	if strings.Index(address, ":") != -1 {
		netProto = "ip6:ipv6-icmp"
	}
	ra, err := net.ResolveIPAddr(netProto, address)
	if err != nil {
		return Error, err
	}

	if ip != nil && ip.To4() != nil {
		p.ListenAddr, _ = net.ResolveIPAddr("ip4", ip.To4().String())
	}

	results := make(map[string]*response)
	results[ra.String()] = nil
	p.AddIPAddr(ra)

	onRecv, onIdle, onErr := make(chan *response), make(chan bool), make(chan int)

	p.OnRecv = func(addr *net.IPAddr, t time.Duration) {
		onRecv <- &response{addr: addr, rtt: t}
	}
	p.OnIdle = func() {
		onIdle <- true
	}

	p.OnErr = func(addr *net.IPAddr, t int) {
		onErr <- t
	}

	p.MaxRTT = time.Second
	go p.Run()

	ret := NoReply
	select {
	case <-onRecv:
		ret = EchoReply
	case <-onIdle:
		ret = NoReply
	case res := <-onErr:
		errID := fmt.Sprintf("%d", res)
		err = errors.New(errID)
		ret = Error
	}

	p.Stop()
	return ret, err
}
