package main

import (
	"os"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/quic-go/quic-go"

	"compnet-socket-labs/project/utils"
)

var (
	DefaultServerIP   = "127.0.0.1"
	DefaultServerPort = "6584"
	BufferSize        = 2048
	AppLayerProto     = "lrt-jakarta-2406396584"
)

func ResolveConfig() (string, string) {
	ip := os.Getenv("SERVER_ADDR")
	if ip == "" {
		ip = DefaultServerIP
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultServerPort
	}
	return ip, port
}

func ResolveALPN() string {
	alpn := os.Getenv("ALPN")
	if alpn == "" {
		alpn = AppLayerProto
	}
	return alpn
}

func main() {
	serverIP, serverPort := ResolveConfig()
	alpn := ResolveALPN()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{alpn},
	}

	connection, err := quic.DialAddr(context.Background(), net.JoinHostPort(serverIP, serverPort), tlsConfig, &quic.Config{})
	if err != nil {
		log.Fatalln(err)
	}
	defer connection.CloseWithError(0x0, "No Error")

	packets := []utils.LRTJPIDSPacket{
		{
			LRTJPIDSPacketFixed: utils.LRTJPIDSPacketFixed{
				TransactionId:        1,
				IsNewTrain:           1,
				TrainNumber:          1002,
				EstimatedArrivalHour: 5,
				EstimatedArrivalMinute: 34,
				DestinationLength:    len("Manggarai"),
			},
			Destination: "Manggarai",
		},
		{
			LRTJPIDSPacketFixed: utils.LRTJPIDSPacketFixed{
				TransactionId:        2,
				IsUpdateTrain:        1,
				TrainNumber:          1002,
				EstimatedArrivalHour: 5,
				EstimatedArrivalMinute: 35,
				DestinationLength:    len("Velodrome"),
			},
			Destination: "Velodrome",
		},
	}

	for _, packet := range packets {
		stream, err := connection.OpenStreamSync(context.Background())
		if err != nil {
			log.Fatalln(err)
		}

		encoded := utils.Encoder(packet)
		if _, err := stream.Write(encoded); err != nil {
			_ = stream.Close()
			log.Fatalln(err)
		}

		ack, err := readPacketFromStream(stream)
		if err != nil && err != io.EOF {
			_ = stream.Close()
			log.Fatalln(err)
		}
		fmt.Printf("[quic] ACK received: %+v\n", ack)
		_ = stream.Close()
	}
}

func readPacketFromStream(reader io.Reader) (utils.LRTJPIDSPacket, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(reader, header); err != nil {
		return utils.LRTJPIDSPacket{}, err
	}

	destinationLength := int(header[7])
	destination := make([]byte, destinationLength)
	if _, err := io.ReadFull(reader, destination); err != nil {
		return utils.LRTJPIDSPacket{}, err
	}

	return utils.Decoder(append(header, destination...)), nil
}