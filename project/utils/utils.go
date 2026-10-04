package utils

import (
	"encoding/binary"
)

// Component data types may be modified, but field names must not be changed
type LRTJPIDSPacketFixed struct {
	TransactionId          int
	IsAck                  int
	IsNewTrain             int
	IsUpdateTrain          int
	IsDeleteTrain          int
	IsTrainArriving        int
	IsTrainDeparting       int
	TrainNumber            int
	EstimatedArrivalHour   int
	EstimatedArrivalMinute int
	DestinationLength      int
}

type LRTJPIDSPacket struct {
	LRTJPIDSPacketFixed
	Destination string
}

func Encoder(packet LRTJPIDSPacket) []byte {
	destination := []byte(packet.Destination)
	if len(destination) > 255 {
		destination = destination[:255]
	}

	buffer := make([]byte, 0, 7+len(destination))

	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(packet.TransactionId))
	buffer = append(buffer, header...)

	flags := byte((packet.IsAck&1)<<7 |
		(packet.IsNewTrain&1)<<6 |
		(packet.IsUpdateTrain&1)<<5 |
		(packet.IsDeleteTrain&1)<<4 |
		(packet.IsTrainArriving&1)<<3 |
		(packet.IsTrainDeparting&1)<<2)
	buffer = append(buffer, flags)

	binary.BigEndian.PutUint16(header, uint16(packet.TrainNumber))
	buffer = append(buffer, header...)
	buffer = append(buffer, byte(packet.EstimatedArrivalHour))
	buffer = append(buffer, byte(packet.EstimatedArrivalMinute))
	buffer = append(buffer, byte(len(destination)))
	buffer = append(buffer, destination...)

	return buffer
}

func Decoder(rawMessage []byte) LRTJPIDSPacket {
	if len(rawMessage) < 8 {
		return LRTJPIDSPacket{}
	}

	packet := LRTJPIDSPacket{}
	packet.TransactionId = int(binary.BigEndian.Uint16(rawMessage[0:2]))

	flags := rawMessage[2]
	packet.IsAck = int(flags>>7) & 1
	packet.IsNewTrain = int(flags>>6) & 1
	packet.IsUpdateTrain = int(flags>>5) & 1
	packet.IsDeleteTrain = int(flags>>4) & 1
	packet.IsTrainArriving = int(flags>>3) & 1
	packet.IsTrainDeparting = int(flags>>2) & 1

	packet.TrainNumber = int(binary.BigEndian.Uint16(rawMessage[3:5]))
	packet.EstimatedArrivalHour = int(rawMessage[5])
	packet.EstimatedArrivalMinute = int(rawMessage[6])
	packet.DestinationLength = int(rawMessage[7])

	expectedLength := 8 + packet.DestinationLength
	if len(rawMessage) < expectedLength {
		return LRTJPIDSPacket{}
	}

	packet.Destination = string(rawMessage[8:expectedLength])
	packet.DestinationLength = len(packet.Destination)

	return packet
}
