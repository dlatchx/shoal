package main

import (
	"log"
	"net"
	"strconv"

	"github.com/songgao/water"
	"github.com/songgao/water/waterutil"
)

func setupInterface(in <-chan []byte, out chan<- []byte, ifaceName *string, virtualIP net.IP) {
	// create TUN interface
	iface, err := water.NewTUN(*ifaceName)
	if err != nil {
		log.Fatal(err)
	}

	*ifaceName = iface.Name()
	log.Printf("interface : %s", *ifaceName)

	// bring interface up
	parseCommand("ip link set dev " + *ifaceName + " up")
	parseCommand("ip link set dev " + *ifaceName + " mtu " + strconv.Itoa(BUFFERSIZE))
	parseCommand("ip addr add " + virtualIP.String() + "/0 dev " + *ifaceName)

	// handle outgoing packets (system -> out)
	buffer := make([]byte, BUFFERSIZE)
	go func() {
		for {
			length, err := iface.Read(buffer)
			if err != nil {
				log.Fatal(err)
			}

			packet := make([]byte, length)
			copy(packet, buffer[:length])

			if waterutil.IsIPv6(packet) {
				out <- packet
			}
		}
	}()

	// handle incoming packets (out -> system)
	go func() {
		for {
			_, err = iface.Write(<-in)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
}
