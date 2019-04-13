package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"hash/crc32"
	"net"
	"os"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	pb "github.com/USA-RedDragon/golang-threadserver/protobuf"
)

var verbose = flag.Bool("verbose", false, "Whether to display verbose logs")

func main() {
	var host = flag.String("listen", "127.0.0.1", "The IP to connect to ")
	var port = flag.Int("port", 2323, "The Port to connect to")
	var packets = flag.Int("packets", 100, "The number of packets to send")

	flag.Parse()

	start(*host, *port, *packets)
}

var conn net.Conn
var aesKey []byte
var wg sync.WaitGroup

func start(host string, port int, packetNum int) {
	log("Creating aes key")
	aesKey = make([]byte, 64)
	_, err := rand.Read(aesKey)
	handleError("Error creating AES key", err)

	log("Dialing Sever Socket at %s on port %d", host, port)
	conn, err = net.Dial("udp", fmt.Sprintf("%s:%d", host, port))
	handleError("Error opening UDP Socket", err)

	wg.Add(packetNum)

	for i := 0; i < packetNum; i++ {
		go sendPacket()
	}

	wg.Wait()

	conn.Close()
}

var packetsSent uint32
var packetNumber uint32

func sendPacket() {
	defer wg.Done()
	authPacket := createPacket(pb.Packet_AUTH, aesKey, "1")
	authPacketBytes, err := proto.Marshal(authPacket)
	handleError("Error marshalling AUTH Packet", err)
	packetsSent++
	log("Sending packet %d", packetsSent)
	conn.Write(authPacketBytes)
}

func createPacket(packetType pb.Packet_PacketType, data []byte, playerID string) *pb.Packet {
	packet := &pb.Packet{}
	packet.Type = packetType
	packet.Sequence = packetNumber
	packetNumber++
	if playerID != "" {
		packet.PlayerID = playerID
	}
	packet.Message = data
	packet.Timestamp = ptypes.TimestampNow()

	packet.MessageLength = uint32(len(data))

	crcData, err := proto.Marshal(packet)
	if err != nil {
		log("Error marshalling protobuf for CRC calculation, Swallowing Error: %v", err)
		return nil
	}
	calculatedCrc := crc32.ChecksumIEEE(crcData)
	packet.Crc = calculatedCrc

	return packet
}

func loadServerKey() *rsa.PublicKey {
	log("Loading Server Key")

	privateKeyFile, err := os.Open("../private_key.pem")
	handleError("Failed to open private_key.pem: ", err)

	pemfileinfo, _ := privateKeyFile.Stat()
	size := pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)

	data, _ := pem.Decode([]byte(pembytes))

	defer privateKeyFile.Close()
	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)
	handleError("Failed to open private_key.pem: ", err)

	return &privateKeyImported.PublicKey
}

func log(log string, a ...interface{}) {
	if *verbose {
		fmt.Printf(log+"\n", a...)
	}
}

func handleError(log string, err error) {
	if err != nil {
		fmt.Println(log, err)
		os.Exit(1)
	}
}
