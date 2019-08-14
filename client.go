package main

import (
	"fmt"
	"net"
)

func handleClientRequest(client net.Conn, intChan chan int, serveraddr string) {
	var msg [msg_max_len]byte
	intChan <- 1
	defer client.Close()

	var b [1024]byte
	_, err := client.Read(b[:])
	if err != nil {
		goto out
	}

	if b[0] == 0x05 { //只处理Socks5协议
		//客户端回应：Socks服务端不需要验证方式
		client.Write([]byte{0x05, 0x00})
		n, err := client.Read(b[:])

		server, err := net.Dial("tcp", serveraddr)
		if err != nil {
			goto out
		}
		defer server.Close()
		msg_len := DoEncap(msg[:], b[0:n], client_header[:])
		server.Write(msg[0:msg_len])
		server.Read(b[:])
		if b[11] == 0x81 {
			client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) //响应客户端连接成功
			//进行转发
			ClientSide(client, server)
		} else {
			client.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		}
	}
out:
	intChan <- -1
	return
}

func connects_static(intChan chan int) {
	routines_num := 0
	for {
		temp_num := <-intChan
		routines_num = routines_num + temp_num
		fmt.Printf("routines_num %d\n", routines_num)
	}
}

func ClientDaemon(clientaddr string, serveraddr string) {

	intChan := make(chan int)
	go connects_static(intChan)
	l, err := net.Listen("tcp", clientaddr)
	if err != nil {
		fmt.Println("create socket failed")
		return
	}

	for {
		client, err := l.Accept()
		if err != nil {
			continue
		}

		go handleClientRequest(client, intChan, serveraddr)
	}
}
