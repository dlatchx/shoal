package main

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Broadcaster struct {
	addrs         []*net.UDPAddr
	addrsMutex    *sync.RWMutex
	announceMutex *sync.RWMutex
	ifaceName     string
}

func NewBroadcaster(ifaceName string) *Broadcaster {
	b := &Broadcaster{make([]*net.UDPAddr, 0), &sync.RWMutex{}, &sync.RWMutex{}, ifaceName}

	go func() {
		for {
			b.UpdateBroadcastIPs()
			time.Sleep(3 * time.Second)
		}
	}()

	return b
}

func (b *Broadcaster) UpdateBroadcastIPs() {
	multicastIP := net.IPv6linklocalallnodes // ff02::1

	b.addrsMutex.Lock()

	b.addrs = make([]*net.UDPAddr, 0)

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}

	for _, iface := range ifaces {
		multiAddrs, err := iface.Addrs()
		if err != nil {
			log.Fatal(err)
		}

		linkLocal := false
		for _, multiAddr := range multiAddrs {
			if multiAddr.String()[:4] == "fe80" {
				linkLocal = true
				break
			}
		}

		if linkLocal && iface.Name != b.ifaceName && (iface.Flags&net.FlagMulticast) == net.FlagMulticast {
			b.addrs = append(b.addrs, &net.UDPAddr{multicastIP, PORT_MULTICAST, iface.Name})
		}
	}

	b.addrsMutex.Unlock()
}

func (b *Broadcaster) Broadcast(msg []byte) {
	b.announceMutex.Lock()
	b.addrsMutex.RLock()

	for _, addr := range b.addrs {
		sendMulti, err := net.DialUDP("udp6", nil, addr)
		if err != nil {
			if strings.Contains(err.Error(), "cannot assign requested address") || strings.Contains(err.Error(), "network is unreachable") {
				continue
			} else {
				log.Fatal(err)
			}
		}
		sendMulti.Write(msg)
		sendMulti.Close()
	}

	// unlock addrs before, so that addr updates
	// have priority over broadcasts
	b.addrsMutex.RUnlock()
	b.announceMutex.Unlock()
}
