package main

import (
	"hash/crc32"
	"net"

	pb "github.com/clevyr/guessmatch-threadserver/protobuf"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

var packetNumber uint32

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

func sendResponses(server *net.UDPConn) {
	pubsub := redisClient.Subscribe("queue:packetreplies")
	for {
		response, err := pubsub.ReceiveMessage()
		if err != nil {
			log("Failed to get redis message: %v", err)
		} else {
			packet := &pb.Packet{}
			err := proto.Unmarshal([]byte(response.Payload), packet)
			if err != nil {
				log("Error parsing response protobuf, Swallowing Error: %v", err)
				continue
			}
			playerMeta := &pb.PlayerMetaData{}
			playerMetaRedis, err := redisClient.Get(packet.PlayerID).Bytes()
			err = proto.Unmarshal(playerMetaRedis, playerMeta)
			if err != nil {
				log("Error parsing response player metadata, Swallowing Error: %v", err)
				continue
			}
			packetBytes, err := proto.Marshal(packet)
			if err != nil {
				log("Error marshalling protobuf for response, Swallowing Error: %v", err)
				return
			}
			remoteAddr := net.UDPAddr{
				IP:   net.ParseIP(playerMeta.Address),
				Port: int(playerMeta.Port),
			}
			_, err = server.WriteToUDP(packetBytes, &remoteAddr)
			if err != nil {
				log("Couldn't send response, Swallowing error: %v", err)
			}
			log("sent %d bytes to %s:%d", len(packetBytes), playerMeta.Address, playerMeta.Port)
		}
	}
}

func checkCRC(packet *pb.Packet) bool {
	crc := packet.GetCrc()
	packet.Crc = 0

	crcData, err := proto.Marshal(packet)
	if err != nil {
		log("Error marshalling protobuf for CRC, Swallowing Error: %v", err)
		return false
	}
	calculatedCrc := crc32.ChecksumIEEE(crcData)
	packet.Crc = crc

	if crc != calculatedCrc {
		log("CRC Doesn't Match: %d %d", crc, calculatedCrc)
		return false
	}

	return true
}
