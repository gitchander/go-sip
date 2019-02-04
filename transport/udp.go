package transport

import (
	"net"
	"sync"
	"time"
)

const maxBufferSize = 4096

type TransportUDP struct {
	udpConn *net.UDPConn
	isOpen  *SyncBool

	cm *connectionsManagerUDP
}

func OpenTransportUDP(address string) (*TransportUDP, error) {

	laddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}

	t := &TransportUDP{
		udpConn: udpConn,
		isOpen:  NewSyncBool(true),
		cm:      newConnectionsManagerUDP(),
	}

	go readDataPacket(t.isOpen, t.udpConn, t.cm)

	return t, nil
}

func (t *TransportUDP) Close() error {

	return nil
}

func readDataPacket(isOpen *SyncBool, udpConn *net.UDPConn, cm *connectionsManagerUDP) {
	defer udpConn.Close()

	buf := make([]byte, maxBufferSize)

	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		checkError(err)

		if !(isOpen.Get()) {
			break
		}

		packet := cloneBytes(buf[:n])
		cm.AddPacket(addr, packet)
	}
}

type udpConn struct {
	addr        *net.UDPAddr
	readPackets chan []byte
	lastTime    time.Time
}

func newUdpConn(addr *net.UDPAddr) *udpConn {
	return &udpConn{
		addr:        addr,
		readPackets: make(chan []byte),
		lastTime:    time.Now(),
	}
}

func (c *udpConn) receivePacket(packet []byte) {
	c.lastTime = time.Now()
	c.readPackets <- packet
}

//----------------------------------------------
// !!!

func (c *udpConn) ReadPacket() (packet []byte) {
	packet = <-c.readPackets
	return packet
}

func (c *udpConn) WritePacket(udpConn *net.UDPConn, packet []byte) error {
	_, err := udpConn.WriteToUDP(packet, c.addr)
	return err
}

// !!!
//----------------------------------------------

type connectionsManagerUDP struct {
	mutex       sync.Mutex
	connections map[string]*udpConn
}

func newConnectionsManagerUDP() *connectionsManagerUDP {
	return &connectionsManagerUDP{
		connections: make(map[string]*udpConn),
	}
}

func (cm *connectionsManagerUDP) getConn(address string) *udpConn {
	cm.mutex.Lock()
	conn := cm.connections[address]
	cm.mutex.Unlock()
	return conn
}

func (cm *connectionsManagerUDP) AddPacket(addr *net.UDPAddr, packet []byte) {
	cm.mutex.Lock()

	key := addr.String()
	c, ok := cm.connections[key]
	if !ok {
		c = newUdpConn(addr)
		cm.connections[key] = c
	}
	c.receivePacket(packet)

	cm.mutex.Unlock()
}

func (cm *connectionsManagerUDP) deleteOlderThan(dur time.Duration) {
	cm.mutex.Lock()
	now := time.Now()
	var keys []string
	for key, conn := range cm.connections {
		if now.Sub(conn.lastTime) > dur {
			keys = append(keys, key)
		}
	}
	for _, key := range keys {
		delete(cm.connections, key)
	}
	cm.mutex.Unlock()
}

func deleteOldConnections(cm *connectionsManagerUDP) {
	dur := 10 * time.Minute
	for {
		time.Sleep(5 * time.Minute)
		cm.deleteOlderThan(dur)
	}
}
