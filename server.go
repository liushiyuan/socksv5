package main

import (
	"fmt"
	"net"
	"strconv"
)

func handleServerRequest(client net.Conn) {
	defer client.Close()

	var b [1300]byte
	n, err := client.Read(b[:])
	if err != nil {
		return
	}
	ToPlain(b[header_len:n])
	var host, port string
	switch b[header_len+3] {
	case 0x01: //IP V4
		host = net.IPv4(b[header_len+4], b[header_len+5], b[header_len+6], b[header_len+7]).String()
	case 0x03: //域名
		host = string(b[header_len+5 : n-2]) //b[4]表示域名的长度
	case 0x04: //IP V6
		host = net.IP{b[header_len+4], b[header_len+5], b[header_len+6], b[header_len+7], b[header_len+8], b[header_len+9], b[header_len+10], b[header_len+11], b[header_len+12], b[header_len+13], b[header_len+14], b[header_len+15], b[header_len+16], b[header_len+17], b[header_len+18], b[header_len+19]}.String()
	}
	port = strconv.Itoa(int(b[n-2])<<8 | int(b[n-1]))

	server, err := net.Dial("tcp", net.JoinHostPort(host, port))
	var ret [1]byte
	if err != nil {
		ret[0] = 0x02
		dst_len := DoEncap(b[:], ret[0:1], server_header[:])
		client.Write(b[0:dst_len])
		return
	}
	defer server.Close()
	ret[0] = 0x01
	dst_len := DoEncap(b[:], ret[0:1], server_header[:])
	client.Write(b[0:dst_len])

	ServerSide(client, server)
	return
}

func ServerDaemon(serveraddr string) {
	l, err := net.Listen("tcp", serveraddr)
	if err != nil {
		fmt.Println("create socket failed")
		return
	}

	for {
		client, err := l.Accept()
		if err != nil {
			continue
		}

		go handleServerRequest(client)
	}
}
