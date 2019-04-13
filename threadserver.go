package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"time"

	pb "github.com/USA-RedDragon/golang-threadserver/protobuf"

	"github.com/go-redis/redis"
	"github.com/golang/protobuf/proto"
)

var redisClient *redis.Client

var packetTypesToChannel = map[pb.Packet_PacketType]string{
	pb.Packet_GAME:          "gamemessages",
	pb.Packet_DATA:          "datamessages",
	pb.Packet_INIT:          "initmessages",
	pb.Packet_RETRY:         "retrymessages",
	pb.Packet_CHECK_TIMEOUT: "checktimeoutmessages",
}

func start(host string, port int, redisHost string) {
	log("Connecting to redis")

	redisClient = redis.NewClient(&redis.Options{
		Addr: redisHost,
	})

	_, err := redisClient.Ping().Result()
	handleError("Failed to connect to redis: ", err)

	for _, element := range packetTypesToChannel {
		redisClient.Publish("queue:"+element, "")
	}

	/*var serverKey *rsa.PrivateKey
	if _, err := os.Stat("private_key.pem"); err == nil {
		serverKey = loadServerKey()
	} else if os.IsNotExist(err) {
		serverKey = generateServerKey()
	}*/

	buffer := make([]byte, 4096)
	sockerAddr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(host),
	}
	log("Opening Server Socket at %s on port %d", host, port)
	server, err := net.ListenUDP("udp", &sockerAddr)
	handleError("Error opening UDP Socket", err)

	go sendResponses(server)

	for {
		len, remoteaddr, err := server.ReadFromUDP(buffer)
		fmt.Printf("Read a message from %v\n", remoteaddr)
		if err != nil {
			log("Error reading from UDP Socket, Swallowing Error: %v", err)
			continue
		}
		go handlePacket(server, remoteaddr, buffer[:len])
	}
}

func handlePacket(server *net.UDPConn, remoteaddr *net.UDPAddr, packetData []byte) {
	packet := &pb.Packet{}
	err := proto.Unmarshal(packetData, packet)
	if err != nil {
		log("Error parsing protobuf, Swallowing Error: %v", err)
		return
	}
	if packet.Type != pb.Packet_HEARTBEAT {
		log(packet.String())
	}

	if !checkCRC(packet) {
		return
	}

	switch packet.Type {
	case pb.Packet_HEARTBEAT:
		response := createPacket(pb.Packet_HEARTBEAT, []byte("Pong"), "")
		if response == nil {
			return
		}
		responseBytes, err := proto.Marshal(response)
		if err != nil {
			log("Error marshalling protobuf for response, Swallowing Error: %v", err)
			return
		}
		_, err = server.WriteToUDP(responseBytes, remoteaddr)
		handleError("Couldn't send heartbeat response: ", err)
	case pb.Packet_AUTH:
		decryptedAESKey := packet.Message
		log("Decrypted AES key: %s", hex.EncodeToString(decryptedAESKey))
		playerMetaData := &pb.PlayerMetaData{}
		playerMetaData.Aes = decryptedAESKey
		playerMetaData.Address = remoteaddr.IP.String()
		playerMetaData.Port = uint32(remoteaddr.Port)
		log("Player ID Auth: %s", packet.PlayerID)
		playerMetaDataBytes, err := proto.Marshal(playerMetaData)
		if err != nil {
			log("Error marshalling PlayerMetaData for redis, Swallowing Error: %v", err)
			return
		}
		redisClient.Set(packet.PlayerID, playerMetaDataBytes, 0)
		response := createPacket(pb.Packet_HEARTBEAT, []byte("Pong"), "")
		responseBytes, err := proto.Marshal(response)
		if err != nil {
			log("Error marshalling protobuf for response, Swallowing Error: %v", err)
			return
		}
		_, err = server.WriteToUDP(responseBytes, remoteaddr)
		handleError("Couldn't send heartbeat response: ", err)
	default:
		if val, ok := packetTypesToChannel[packet.Type]; ok {
			packetBytes, err := proto.Marshal(packet)
			if err != nil {
				log("Error marshalling packet for redis, Swallowing Error: %v", err)
				return
			}
			redisClient.LPush("queue:"+val+":list", packetBytes)
			redisClient.Publish("queue:"+val, "")
			redisClient.Set("active:"+packet.PlayerID, "", 2*time.Minute)
		}
	}

}
