package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"log/slog"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	raddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:42069")
	if err != nil { logger.Error(err.Error()) }

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil { logger.Error(err.Error()) }
	defer conn.Close()
	
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(">")
		line, err := reader.ReadString('\n')
		if err != nil { logger.Error(err.Error()) }
		conn.Write([]byte(line))
	}
}
