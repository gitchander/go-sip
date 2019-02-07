package main

import (
	"fmt"
	"log"

	"github.com/gitchander/go-sip/transport"
)

func main() {
	_, err := transport.OpenTransportUDP(":8050", myHandler{})
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type myHandler struct{}

func (myHandler) Handle(c transport.Conn) {

	for {
		packet, err := c.ReadPacket()
		if err != nil {
			log.Println("ERROR:", err)
			return
		}

		fmt.Printf("read packet: [%X]\n", packet)

		n, err := c.WritePacket(packet)
		_ = n
		//fmt.Println(">>", n, len(packet))
		if err != nil {
			log.Println("ERROR:", err)
			return
		}
	}
}

/*

Concepth:


conn - Connection {Addr}

conn.Read() -> packet -> (decode message, sip-invite)

make answer ...

// encode answer to packet
conn.Write(packet)









*/
