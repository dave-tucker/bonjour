package bonjour

import (
	"log"
	"net"
	"os"
	"time"
)

// Notify is used to notify watchers when a member has been added or removed
type Notify interface {
	NewMember(net.IP)
	RemoveMember(net.IP)
}

// Bonjour is daemon that will advertise a given service and notify when others join or leave
type Bonjour struct {
	ServiceName   string
	ServiceDomain string
	ServicePort   int
	InterfaceName string
	BindToIntf    bool
	Notify        Notify
}

type cacheEntry struct {
	serviceEntry *ServiceEntry
	lastSeen     time.Time
}

var dnsCache map[string]cacheEntry
var queryChan chan *ServiceEntry

func (b Bonjour) publishOnce() {
	ifName := b.InterfaceName
	var iface *net.Interface
	var err error
	if ifName != "" {
		iface, err = net.InterfaceByName(ifName)
		if err != nil {
			log.Println(err.Error())
		}
	}
	instance, err := os.Hostname()
	_, err = Register(instance, b.ServiceName,
		b.ServiceDomain, b.ServicePort,
		[]string{"txtv=1", "key1=val1", "key2=val2"}, iface, b.BindToIntf)
	if err != nil {
		log.Println(err.Error())
	}
}

func (b Bonjour) publish() {
	sleeper := time.Second * 30
	for {
		b.publishOnce()
		time.Sleep(sleeper)
	}
}

func (b Bonjour) lookup(resolver *Resolver, query chan *ServiceEntry) {
	for {
		select {
		case e := <-query:
			err := resolver.Lookup(e.Instance, e.Service, e.Domain)
			if err != nil {
				log.Println("Failed to browse:", err.Error())
			}
		}
	}
}

func (b Bonjour) resolve(resolver *Resolver, results chan *ServiceEntry) {
	err := resolver.Browse(b.ServiceName, b.ServiceDomain)
	if err != nil {
		log.Println("Failed to browse:", err.Error())
	}
	for e := range results {
		if e.AddrIPv4 == nil {
			queryChan <- e
		} else if !isMyAddress(e.AddrIPv4.String()) {
			if e.TTL > 0 {
				if _, ok := dnsCache[e.AddrIPv4.String()]; !ok {
					log.Printf("New Bonjour Member : %s, %s, %s, %s",
						e.Instance, e.Service, e.Domain, e.AddrIPv4)
					b.publishOnce()
					if b.Notify != nil {
						b.Notify.NewMember(e.AddrIPv4)
					}
				}
				dnsCache[e.AddrIPv4.String()] = cacheEntry{e, time.Now()}
			} else {
				log.Printf("Bonjour Member Gone : %s, %s, %s, %s", e.Instance, e.Service, e.Domain, e.AddrIPv4)
				if b.Notify != nil {
					b.Notify.RemoveMember(e.AddrIPv4)
				}
				delete(dnsCache, e.AddrIPv4.String())
			}
		}
	}
}

func isMyAddress(address string) bool {
	intAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range intAddrs {
		if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.String() == address {
			return true
		}
	}
	return false
}

// IsInterfaceEligible checks if an interface is eligible to bind to
func IsInterfaceEligible(bIntf *net.Interface) bool {
	if bIntf.Flags&net.FlagLoopback == 0 {
		addrs, err := bIntf.Addrs()
		if err != nil {
			return false
		}
		for i := 0; i < len(addrs); i++ {
			ip, _, err := net.ParseCIDR(addrs[i].String())
			if err == nil && ip.To4() != nil {
				ret, err := echo("224.0.0.1", &ip)
				if err == nil && ret == EchoReply {
					return true
				}
			}
			// TODO : Handle IPv6
		}
	}
	return false
}

// EligibleInterfacesToBind returns a list of interfaces that can be bound to
func EligibleInterfacesToBind() []*net.Interface {
	var eligibleIfaces = []*net.Interface{}
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, bIntf := range ifaces {
			if IsInterfaceEligible(&bIntf) {
				eligibleIfaces = append(eligibleIfaces, &bIntf)
			}
		}
	}
	return eligibleIfaces
}

// InterfaceToBind returns the first interface that can be bound to
func InterfaceToBind() *net.Interface {
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, bIntf := range ifaces {
			if IsInterfaceEligible(&bIntf) {
				return &bIntf
			}
		}
	}
	return nil
}

func (b Bonjour) keepAlive(resolver *Resolver) {
	sleeper := time.Second * 30
	for {
		for key, e := range dnsCache {
			if time.Now().Sub(e.lastSeen) > sleeper*2 {
				if b.Notify != nil {
					b.Notify.RemoveMember(net.ParseIP(key))
				}
				delete(dnsCache, key)
				log.Println("Bonjour Member timed out : ", key)
			}
		}
		time.Sleep(sleeper)
	}
}

// Start will start the Bonjour daemon
func (b Bonjour) Start() error {
	dnsCache = make(map[string]cacheEntry)
	queryChan = make(chan *ServiceEntry)
	results := make(chan *ServiceEntry)
	resolver, err := NewResolver(nil, results)
	if err != nil {
		log.Println("Failed to initialize resolver:", err.Error())
		return err
	}

	go b.publish()
	go b.resolve(resolver, results)
	go b.lookup(resolver, queryChan)
	go b.keepAlive(resolver)
	return nil
}
