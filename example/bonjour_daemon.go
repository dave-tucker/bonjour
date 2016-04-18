package main

import (
	"fmt"
	"net"
	"os"

	"github.com/socketplane/bonjour"
)

const dockerClusterService = "_foobar._service"
const dockerClusterDomain = "local"

func main() {
	var intfName = ""
	if len(os.Args) > 1 {
		intfName = os.Args[1]
	}
	b := bonjour.Bonjour{
		ServiceName:   dockerClusterService,
		ServiceDomain: dockerClusterDomain,
		ServicePort:   9999,
		InterfaceName: intfName,
		BindToIntf:    true,
		Notify:        notify{},
	}
	b.Start()

	select {}
}

type notify struct{}

func (n notify) NewMember(addr net.IP) {
	fmt.Println("New Member Added : ", addr)
}
func (n notify) RemoveMember(addr net.IP) {
	fmt.Println("Member Left : ", addr)
}
