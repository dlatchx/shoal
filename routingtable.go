package main

import (
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

type RoutingRule struct {
	Distance uint8
	NextHop  *net.UDPAddr
	Time     time.Time
	Sequence uint16
	Public   bool
}

type RoutingTable struct {
	routes      map[string][]*RoutingRule
	virtualIP   net.IP
	mutex       *sync.RWMutex
	sequence    uint16
	broadcaster *Broadcaster
}

func NewRoutingTable(ip net.IP, broadcaster *Broadcaster) *RoutingTable {
	rt := &RoutingTable{make(map[string][]*RoutingRule), ip, &sync.RWMutex{}, 0, broadcaster}

	go func() {
		for {
			rt.AnnounceAll()
			time.Sleep(time.Second / 15)
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100))) // avoid jitter
		}
	}()

	go rt.deleteOldRoutesJob()

	if *debug_showRoutes {
		go func() {
			for {
				rt.printRoutes()
			}
		}()
	}

	return rt
}

func (rt *RoutingTable) printRoutes() {
	rt.mutex.RLock()
	log.Printf("+----------+-----+-------+---+----------------------------+--------+")
	log.Printf("|   dest   | hop |  seq  | p |          next hop          | iface  |")
	log.Printf("+----------+-----+-------+---+----------------------------+--------+")
	for ip, rules := range rt.routes {
		if len(rules) > 0 {
			for i, rule := range rules {
				pub := ' '
				if rule.Public {
					pub = 'X'
				}
				if i == 0 {
					log.Printf("| %8s | %3d | %5d | %c | %26s |%7s |", net.IP(ip).String(), rule.Distance, rule.Sequence, pub, rule.NextHop.IP.String(), rule.NextHop.Zone)
				} else {
					log.Printf("|          | %3d | %5d | %c | %26s |%7s |", rule.Distance, rule.Sequence, pub, rule.NextHop.IP.String(), rule.NextHop.Zone)
				}
			}
			log.Printf("+----------+-----+-------+---+----------------------------+--------+")
		}
	}
	rt.mutex.RUnlock()
	log.Printf("\n")

	time.Sleep(time.Second / 2)
}

func (rt *RoutingTable) NextHop(ip net.IP) *net.UDPAddr {
	rt.mutex.RLock()

	var nextHop *net.UDPAddr
	nextHop = nil

	routingRules := rt.routes[string(ip)]
	if routingRules != nil && len(routingRules) > 0 {
		nextHop = routingRules[0].NextHop
		bestPath := byte(255)
		for _, rule := range routingRules {
			if rule.Distance < bestPath {
				bestPath = rule.Distance
				nextHop = rule.NextHop
			}
		}
	}

	if nextHop == nil {
		rt.mutex.RUnlock()
		return nil
	} else {
		result := &net.UDPAddr{}
		*result = *nextHop

		rt.mutex.RUnlock()
		return result
	}
}

func (rt *RoutingTable) SetRule(distance uint8, nextHop *net.UDPAddr, vIP net.IP, sequence uint16) {
	rt.mutex.Lock()
	if rt.routes[string(vIP)] == nil {
		rules := make([]*RoutingRule, 1)
		rules[0] = &RoutingRule{distance, nextHop, time.Now(), sequence, true}
		rt.routes[string(vIP)] = rules

		msg := make([]byte, 50)
		msg[1] = distance
		copy(msg[2:18], vIP)
		copy(msg[18:34], rt.virtualIP)
		binary.LittleEndian.PutUint16(msg[34:50], sequence)
		rt.broadcaster.Broadcast(msg)
	} else {
		add := true
		for _, rule := range rt.routes[string(vIP)] {
			if rule.NextHop.IP.Equal(nextHop.IP) {
				add = false

				rule.Public = (rule.Sequence <= uint16(65515) && sequence > rule.Sequence) || ((rule.Sequence > uint16(65515) && sequence < uint16(20)) || sequence > rule.Sequence) || (sequence == rule.Sequence && distance < rule.Distance)

				rule.Distance = distance
				rule.Time = time.Now()
				rule.Sequence = sequence

				break
			}
		}
		if add {
			rt.routes[string(vIP)] = append(rt.routes[string(vIP)], &RoutingRule{distance, nextHop, time.Now(), sequence, true})
		}
	}
	rt.mutex.Unlock()
}

func (rt *RoutingTable) AnnounceAll() {
	// announce myself
	msg := make([]byte, 50)
	copy(msg[2:18], rt.virtualIP)
	copy(msg[18:34], rt.virtualIP)

	binary.LittleEndian.PutUint16(msg[34:50], rt.sequence)
	rt.sequence++
	rt.broadcaster.Broadcast(msg)

	// announce my other routes
	msg2 := make([]byte, 50)
	copy(msg2[18:34], rt.virtualIP)

	rt.mutex.RLock()
	for ip, rules := range rt.routes {
		copy(msg2[2:18], net.IP(ip))
		bestPath := byte(255)
		var bestIndex int
		for i, rule := range rules {
			if rule.Distance < bestPath {
				bestPath = rule.Distance
				bestIndex = i
			}
		}
		if bestPath < 255 && rules[bestIndex].Public {
			msg2[1] = bestPath
			binary.LittleEndian.PutUint16(msg2[34:50], rules[bestIndex].Sequence)
			rt.broadcaster.Broadcast(msg2)
		}
	}
	rt.mutex.RUnlock()
}

func (rt *RoutingTable) deleteOldRoutesJob() {
	pause := true
	for {
		if pause {
			rt.mutex.Lock()
		}
		pause = true

		for ip, rules := range rt.routes {
			for i, rule := range rules {
				if time.Now().After(rule.Time.Add(3 * time.Second / 15)) {
					rt.routes[ip] = append(rt.routes[ip][:i], rt.routes[ip][i+1:]...)
					pause = false
					break
				}
			}
		}

		if pause {
			rt.mutex.Unlock()
			time.Sleep(time.Second / 5)
		}
	}
}
