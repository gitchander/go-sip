package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

func main() {
	address := "localhost:8050"
	n := 10
	wg := new(sync.WaitGroup)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go testClient(wg, i, address)
	}
	wg.Wait()
}

func testClient(wg *sync.WaitGroup, number int, address string) {

	defer wg.Done()

	//Connect udp
	conn, err := net.Dial("udp", address)
	checkError(err)
	defer conn.Close()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	const headerSize = 4

	for i := 0; i < 10; i++ {

		payloadSize := r.Intn(2048)
		packetSize := headerSize + payloadSize

		//fmt.Println(nn)
		packet := make([]byte, packetSize)
		binary.BigEndian.PutUint32(packet, uint32(number))

		payload := packet[headerSize:]
		for i := range payload {
			payload[i] = byte(r.Intn(256))
		}

		//simple write
		_, err = conn.Write(packet)
		checkError(err)

		//simple Read
		receiveData := make([]byte, 100) // packetSize)
		n, err := conn.Read(receiveData)
		checkError(err)

		packetReceive := receiveData[:n]

		if !(bytes.Equal(packet, packetReceive)) {
			//fmt.Printf("sent [% X]\n", packet)
			//fmt.Printf("recv [% X]\n", packetReceive)

			fmt.Printf("sent(%d)-recv(%d)\n", len(packet), len(packetReceive))
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
