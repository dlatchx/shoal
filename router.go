package main

import (
	"encoding/binary"
	"log"
	"net"
	"strconv"
)

func router(in chan<- []byte, out chan []byte, virtualIP net.IP, ifaceName string) {
	parseCommand("sysctl net.ipv4.icmp_echo_ignore_broadcasts=0")

	broadcaster := NewBroadcaster(ifaceName)
	routingTable := NewRoutingTable(virtualIP, broadcaster)

	multicastIP := net.IPv6linklocalallnodes // ff02::1
	multicastAddr := &net.UDPAddr{multicastIP, PORT_MULTICAST, ""}

	// listen multicast
	listenMulti, err := net.ListenMulticastUDP("udp6", nil, multicastAddr)
	if err != nil {
		log.Fatal(err)
	}

	listenMulti.SetReadBuffer(MAXDATAGRAMSIZE)
	go func() {
		buffer := make([]byte, MAXDATAGRAMSIZE)
		for {
			length, src, err := listenMulti.ReadFromUDP(buffer)
			if err != nil {
				log.Fatal("ReadFromUDP failed:", err)
			}
			src.Port = PORT_UNICAST

			if buffer[0] == 0 && length == 50 {
				vIP := net.IP(buffer[2:18])
				senderIP := net.IP(buffer[18:34])
				if vIP.Equal(virtualIP) || senderIP.Equal(virtualIP) {
					continue
				}

				sequence := binary.LittleEndian.Uint16(buffer[34:50])

				//log.Printf("%s can join %s with %d hops (sequence %d)", src.IP.String(), vIP.String(), buffer[1], sequence)
				routingTable.SetRule(buffer[1]+1, src, vIP, sequence)
			}
		}
	}()

	// handle outgoing packets
	go func() {
		for {
			outPacket := <-out
			//log.Printf("%s -> %s", IPSrc(outPacket).String(), IPDst(outPacket).String())
			nextHop := routingTable.NextHop(IPDst(outPacket))
			if nextHop != nil {
				//log.Printf("%s -> %s", IPSrc(outPacket).String(), IPDst(outPacket).String())
				socket, err := net.DialUDP("udp6", nil, nextHop)
				if err != nil {
					log.Fatal(err)
				}

				socket.SetWriteBuffer(MAXDATAGRAMSIZE)
				socket.Write(outPacket)
				socket.Close()
			} else {
				//log.Printf("%s -> %s (dropped)", IPSrc(outPacket).String(), IPDst(outPacket).String())
			}
		}
	}()

	unicastAddr, err := net.ResolveUDPAddr("udp6", ":"+strconv.Itoa(PORT_UNICAST))
	if err != nil {
		log.Fatal(err)
	}

	// listen unicast
	listenUni, err := net.ListenUDP("udp6", unicastAddr)
	if err != nil {
		log.Fatal(err)
	}
	listenUni.SetReadBuffer(MAXDATAGRAMSIZE)
	buffer := make([]byte, MAXDATAGRAMSIZE)
	for {
		length, _, err := listenUni.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal("ReadFromUDP failed:", err)
		}

		packet := make([]byte, length)
		copy(packet, buffer[:length])

		packet[7]--

		//log.Printf("%s <- %s", IPDst(packet).String(), IPSrc(packet).String())

		if IPDst(packet).Equal(virtualIP) {
			in <- packet
		} else if packet[7] > 0 {
			out <- packet
		} //else {
		// TODO : send ICMP "Time Exceeded", "hop limit exceeded in transit"
		//}
	}
}
