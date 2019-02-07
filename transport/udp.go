package transport

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const maxBufferSize = 4096

type Conn interface {
	ReadPacket() ([]byte, error)
	WritePacket([]byte) (n int, err error)
}

type ConnectHandler interface {
	Handle(Conn)

	//isHandler()
}

type TransportUDP struct {
	udpConn *net.UDPConn
	isOpen  *SyncBool

	cm *connectionsManagerUDP
}

func OpenTransportUDP(address string, handler ConnectHandler) (*TransportUDP, error) {

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
		cm:      newConnectionsManagerUDP(udpConn, handler),
	}

	go deleteOldConnections(t.cm)

	// work
	loopReadPacket(t.isOpen, t.udpConn, t.cm)

	return t, nil
}

func (t *TransportUDP) Close() error {

	return nil
}

func loopReadPacket(isOpen *SyncBool, udpConn *net.UDPConn, cm *connectionsManagerUDP) {
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

type packetUDP struct {
	addr *net.UDPAddr
	data []byte
	size int
}

func loopWritePacket(udpConn *net.UDPConn, packets <-chan *packetUDP) {
	for {
		p, ok := <-packets
		if !ok {
			break
		}

		_, err := udpConn.WriteToUDP(p.data[:p.size], p.addr)
		checkError(err)

	}
}

type udpConn struct {
	conn        *net.UDPConn
	addr        *net.UDPAddr
	readPackets chan []byte
	lastTime    time.Time
}

func newUdpConn(conn *net.UDPConn, addr *net.UDPAddr) *udpConn {
	return &udpConn{
		conn:        conn,
		addr:        addr,
		readPackets: make(chan []byte),
		lastTime:    time.Now(),
	}
}

func (c *udpConn) Close() error {
	fmt.Println("close conn:", c.addr)
	return nil
}

func (c *udpConn) receivePacket(packet []byte) {
	c.lastTime = time.Now()
	c.readPackets <- packet
}

//----------------------------------------------
// !!!

func (c *udpConn) ReadPacket() ([]byte, error) {
	packet, ok := <-c.readPackets
	if ok {
		return packet, nil
	}
	return nil, fmt.Errorf("close connection (%s)", c.addr)
}

func (c *udpConn) WritePacket(packet []byte) (n int, err error) {
	return c.conn.WriteToUDP(packet, c.addr)
}

// !!!
//----------------------------------------------

type connectionsManagerUDP struct {
	mutex       sync.Mutex
	connections map[string]*udpConn

	conn    *net.UDPConn
	handler ConnectHandler
}

func newConnectionsManagerUDP(conn *net.UDPConn, handler ConnectHandler) *connectionsManagerUDP {
	return &connectionsManagerUDP{
		connections: make(map[string]*udpConn),

		conn:    conn,
		handler: handler,
	}
}

//func (cm *connectionsManagerUDP) getConn_(address string) *udpConn {
//	cm.mutex.Lock()
//	conn := cm.connections[address]
//	cm.mutex.Unlock()
//	return conn
//}

func (cm *connectionsManagerUDP) AddPacket(addr *net.UDPAddr, packet []byte) {
	cm.mutex.Lock()

	key := addr.String()
	c, ok := cm.connections[key]
	if !ok {
		c = newUdpConn(cm.conn, addr)
		go cm.handler.Handle(c)
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
		conn, ok := cm.connections[key]
		if ok {
			delete(cm.connections, key)
			conn.Close()
		}
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
