package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/quic-go/quic-go"

	"compnet-socket-labs/project/utils"
)

var (
	DefaultServerIP   = "0.0.0.0"
	DefaultServerPort = "6584"
	ServerType        = "udp4"
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

func Handler(packet utils.LRTJPIDSPacket) string {
	switch {
	case packet.IsNewTrain != 0:
		return fmt.Sprintf("ADD | %d | %s | %02d:%02d", packet.TrainNumber, packet.Destination, packet.EstimatedArrivalHour, packet.EstimatedArrivalMinute)
	case packet.IsUpdateTrain != 0:
		return fmt.Sprintf("UPD | %d | %s | %02d:%02d", packet.TrainNumber, packet.Destination, packet.EstimatedArrivalHour, packet.EstimatedArrivalMinute)
	default:
		return ""
	}
}

func main() {
	serverIP, serverPort := ResolveConfig()
	alpn := ResolveALPN()

	udpAddress, err := net.ResolveUDPAddr(ServerType, net.JoinHostPort(serverIP, serverPort))
	if err != nil {
		log.Fatalln(err)
	}

	socket, err := net.ListenUDP(ServerType, udpAddress)
	if err != nil {
		log.Fatalln(err)
	}
	defer socket.Close()

	tlsConfig := &tls.Config{
		Certificates: utils.GenerateTLSSelfSignedCertificates(),
		NextProtos:   []string{alpn},
	}

	listener, err := quic.Listen(socket, tlsConfig, &quic.Config{})
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()

	fmt.Printf("[%s] Listening on %s (ALPN: %s)\n", ServerType, socket.LocalAddr(), alpn)
	fmt.Printf("Press Ctrl+C or Cmd+C to stop the program\n")

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("[quic] Accept error: %v\n", err)
			continue
		}

		go connectionHandler(conn)
	}
}

func connectionHandler(connection *quic.Conn) {
	fmt.Printf("[quic] Receive connection from %s\n", connection.RemoteAddr())

	for {
		stream, err := connection.AcceptStream(context.Background())
		if err != nil {
			if err != io.EOF {
				log.Printf("[quic] Accept stream error: %v\n", err)
			}
			return
		}

		go streamHandler(stream)
	}
}

func streamHandler(stream *quic.Stream) {
	defer stream.Close()

	packet, err := readPacketFromStream(stream)
	if err != nil {
		if err != io.EOF {
			log.Printf("[quic] Read packet error: %v\n", err)
		}
		return
	}

	response := Handler(packet)
	if response != "" {
		fmt.Println(response)
	}

	ack := packet
	ack.IsAck = 1
	ackBytes := utils.Encoder(ack)
	if _, err := stream.Write(ackBytes); err != nil {
		log.Printf("[quic] Write ACK error: %v\n", err)
		return
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
