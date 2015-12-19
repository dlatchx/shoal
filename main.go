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
	fmt.Printf("usage : shoal [options...] ip/mask \n\n")
	fmt.Printf("options :\n")
	flag.PrintDefaults()
}

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
	virtualIP, virtualIPNet, err := net.ParseCIDR(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	virtualIPNet.IP = virtualIP // replace the block ip by the one provided in flag.Arg(0)
	log.Printf("virtual IP : %s", virtualIPNet.String())

	// connect TUN interface with router core
	in := make(chan []byte)
	out := make(chan []byte)
	ifaceName := setupInterface(in, out, virtualIPNet)
	router(in, out, virtualIP, ifaceName)
}
