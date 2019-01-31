package transport

import (
	"net"
)

const maxBufferSize = 16000

type TransportUDP struct {
	udpConn     *net.UDPConn
	readPackets chan *udpPacket
	isOpen      *SyncBool
}

func OpenTransportUDP(address string) (*TransportUDP, error) {

	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}

	t := &TransportUDP{
		udpConn:     udpConn,
		readPackets: make(chan *udpPacket, 20),
		isOpen:      NewSyncBool(true),
	}

	go readDataPacket(t.isOpen, t.udpConn, t.readPackets)

	return t, nil
}

func (t *TransportUDP) Close() error {

	return nil
}

func (t *TransportUDP) WritePacket(p *udpPacket) error {
	_, err := t.udpConn.WriteToUDP(p.data, p.addr)
	return err
}

func readDataPacket(isOpen *SyncBool, udpConn *net.UDPConn, packets chan<- *udpPacket) {
	defer udpConn.Close()

	buf := make([]byte, maxBufferSize)

	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		checkError(err)

		if !(isOpen.Get()) {
			break
		}

		packet := &udpPacket{
			addr: addr,
			data: cloneBytes(buf[:n]),
		}

		packets <- packet
	}
}

type udpPacket struct {
	addr *net.UDPAddr
	data []byte
}

// type udpConn struct {
// }
