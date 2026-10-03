package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"
)

var (
	DefaultServerIP   = "18.207.183.144"
	DefaultServerPort = "6584"
	ServerType        = "udp4"
	BufferSize        = 2048
	AppLayerProto     = "compnet-quic-sample-aydin"
	LogDir            = "logs"
	SSLKeyLogFileName = "ssl-key.log"
	StreamCount       = 2
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
	keylogFlag := flag.Bool("keylog", false, "Enable TLS key logging to logs/ssl-key.log for Wireshark inspection")
	qlogFlag := flag.Bool("qlog", false, "Enable QLOG event tracing to logs/*.sqlog")
	flag.Parse()

	if *keylogFlag || *qlogFlag {
		if err := os.MkdirAll(LogDir, 0755); err != nil {
			log.Fatalf("failed to create logs directory: %v", err)
		}
	}

	serverIP, serverPort := ResolveConfig()
	alpn := ResolveALPN()

	fmt.Printf("QUIC Client Socket Program Example in Go\n")
	fmt.Printf("[%s] Connecting to %s\n", ServerType, net.JoinHostPort(serverIP, serverPort))

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Self-signed test certificate
		NextProtos:         []string{alpn},
	}

	if *keylogFlag {
		keyLogPath := filepath.Join(LogDir, SSLKeyLogFileName)
		keyLogFile, err := os.OpenFile(keyLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatalf("failed to open keylog file: %v", err)
		}
		defer keyLogFile.Close()
		tlsConfig.KeyLogWriter = keyLogFile
		fmt.Printf("[quic] TLS Key logging enabled -> %s\n", keyLogPath)
	}

	quicConfig := &quic.Config{}
	if *qlogFlag {
		if os.Getenv("QLOGDIR") == "" {
			_ = os.Setenv("QLOGDIR", LogDir)
		}
		quicConfig.Tracer = qlog.DefaultConnectionTracer
		fmt.Printf("[quic] QLOG event tracing enabled -> %s/*.sqlog\n", LogDir)
	}

	connection, err := quic.DialAddr(context.Background(), net.JoinHostPort(serverIP, serverPort), tlsConfig, quicConfig)
	if err != nil {
		log.Fatalln(err)
	}
	defer connection.CloseWithError(0x0, "No Error")

	fmt.Printf("[quic] Dialling from %s to %s\n", connection.LocalAddr(), connection.RemoteAddr())

	fmt.Printf("[quic] Creating receive buffer of size %d\n", BufferSize)

	fmt.Printf("[quic] Input message to be sent to server: ")
	message, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatalln(err)
	}

	streams := make([]*quic.Stream, StreamCount)
	for i := range streams {
		streams[i], err = connection.OpenStreamSync(context.Background())
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("[quic] Opened bidirectional stream %d to %s\n", streams[i].StreamID(), connection.RemoteAddr())
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(streams))
	for _, stream := range streams {
		go func(stream *quic.Stream) {
			defer waitGroup.Done()
			defer stream.Close()

			fmt.Printf("[quic] [Stream %d] Sending message '%s' to server\n", stream.StreamID(), message)
			if _, err := stream.Write([]byte(message)); err != nil {
				log.Printf("[quic] Stream %d write error: %v\n", stream.StreamID(), err)
				return
			}

			streamReceiveBuffer := make([]byte, BufferSize)
			receiveLength, err := stream.Read(streamReceiveBuffer)
			if err != nil && err != io.EOF {
				log.Printf("[quic] Stream %d read error: %v\n", stream.StreamID(), err)
				return
			}

			fmt.Printf("[quic] [Stream %d] Received %d bytes of message from server\n", stream.StreamID(), receiveLength)
			response := string(streamReceiveBuffer[:receiveLength])
			fmt.Printf("[quic] [Stream %d] Response from server: %s\n", stream.StreamID(), response)
		}(stream)
	}
	waitGroup.Wait()
}
