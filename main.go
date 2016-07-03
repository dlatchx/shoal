package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

const (
	BUFFERSIZE      = 65431
	MAXDATAGRAMSIZE = 65431
	PORT_MULTICAST  = 6942
	PORT_UNICAST    = 6943
)

var usage = func() {
	fmt.Printf("usage : shoal [options...] ip \n\n")
	fmt.Printf("options :\n")
	flag.PrintDefaults()
}

var ifaceName = flag.String("interface", "", "virtual interface name")
var debug_showPackets = flag.Bool("showpackets", false, "log every routed packets. may drastically drain performance, use it for debug")
var debug_showRoutes = flag.Bool("showroutes", false, "print routing table every 0.5s. use it for debug")

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	log.Print("starting shoal")

	// parse the node's vitual ip
	virtualIP := net.ParseIP(flag.Arg(0))
	if virtualIP == nil {
		log.Fatal("invalid ip")
	}
	log.Printf("virtual IP : %s", virtualIP.String())

	// connect TUN interface with router core
	in := make(chan []byte)
	out := make(chan []byte)
	setupInterface(in, out, ifaceName, virtualIP)
	router(in, out, virtualIP, *ifaceName)
}
