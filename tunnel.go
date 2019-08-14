package main

import (
	"container/list"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type ENC_MODE int32

const (
	ENCAP           ENC_MODE = 0
	DECAP           ENC_MODE = 1
	CLIENT_SIDE     int      = 0
	SERVER_SIDE     int      = 1
	content_max_len int      = 1000
	header_len      int      = 11
	msg_max_len     int      = content_max_len + header_len
)

var (
	client_header = []byte{0x16, 0x03, 0x03, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x03, 0x03}
	server_header = []byte{0x16, 0x03, 0x03, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x03}
)

func ToEnc(plain []byte) {
	for i := 0; i < len(plain); i++ {
		if plain[i] < 128 {
			plain[i] = plain[i] + 128
		} else {
			plain[i] = plain[i] - 128
		}
	}
}

func ToPlain(enc []byte) {
	for i := 0; i < len(enc); i++ {
		if enc[i] < 128 {
			enc[i] = enc[i] - 128
		} else {
			enc[i] = enc[i] + 128
		}
	}
}

func DoEncap(dst []byte, src []byte, header []byte) int {
	ToEnc(src[:])
	//fmt.Printf("encap read[%d]\n", n)
	//fmt.Println(content[0:n])
	len1 := header[3 : 3+2]
	len2 := header[7 : 7+2]

	binary.BigEndian.PutUint16(len2, uint16(len(src)+2))
	binary.BigEndian.PutUint16(len1, uint16(len(src)+2+4))
	copy(dst[0:len(header)], header[:])
	copy(dst[len(header):], src[:])
	return len(header) + len(src)
}

func copyFromTun(dst net.Conn, src net.Conn) int {
	var msg_cnt [content_max_len]byte
	var msg_hdr [header_len]byte

	n, err := src.Read(msg_hdr[:])
	if err != nil {
		return 1
	}
	if n != header_len {
		remain_len := header_len - n
		n, err = src.Read(msg_hdr[n:])
		if err != nil {
			return 1
		}
		if n != remain_len {
			fmt.Println("from tun header discomplete")
			return 1
		}
	}
	length := int(msg_hdr[3])*256 + int(msg_hdr[4])
	msg_len := length + 5 - header_len
	n, err = src.Read(msg_cnt[0:msg_len])
	if err != nil {
		return 1
	}
	if n != msg_len {
		remain_len := msg_len - n
		n, err = src.Read(msg_cnt[n:msg_len])
		if err != nil {
			return 1
		}
		if n != remain_len {
			fmt.Println("from tun msg discomplete")
			return 1
		}
	}

	ToPlain(msg_cnt[0:msg_len])
	n, err = dst.Write(msg_cnt[0:msg_len])
	if err != nil {
		return 1
	}
	return 0
}

func clientCopy(dst net.Conn, src net.Conn, mode ENC_MODE) {
	var content [content_max_len]byte
	client_hello := []byte{0x16, 0x03, 0x03, 0x00, 0x06, 0x01, 0x00, 0x00, 0x02, 0x03, 0x03}

	var msg [msg_max_len]byte

	for {
		src.SetReadDeadline(time.Now().Add(time.Second * 30))
		if mode == ENCAP {
			n, err := src.Read(content[:])
			if err != nil {
				return
			}

			dst_len := DoEncap(msg[:], content[0:n], client_header[:])
			n, err = dst.Write(msg[0:dst_len])
			if err != nil {
				return
			}
			//fmt.Printf("encap side 0 mode %d write[%d]\n", mode, dst_len)
		} else {
			ret := copyFromTun(dst, src)
			if ret == 1 {
				return
			}
			_, err := src.Write(client_hello[:])
			if err != nil {
				return
			}
		}
	}
}

func serverCopy(dst net.Conn, src net.Conn, mode ENC_MODE, intChan chan int, server_buffer *list.List) {
	var content [content_max_len]byte

	for {
		src.SetReadDeadline(time.Now().Add(time.Second * 30))
		if mode == ENCAP {
			n, err := src.Read(content[:])
			if err != nil {
				return
			}

			temp := make([]byte, n)
			copy(temp[0:n], content[0:n])
			server_buffer.PushBack(temp)
			//fmt.Printf("encap write[%d]\n", len(client_header)+n)
			//fmt.Println(msg[0 : len(client_header)+n])
		} else {
			ret := copyFromTun(dst, src)
			if ret == 1 {
				return
			}
			intChan <- 1
		}
	}
}

func keep_alive(intChan chan int, term *bool) {
	for {
		if *term == true {
			return
		}
		intChan <- 1
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func do_sendmsg(dst net.Conn, server_buffer *list.List, intChan chan int) {
	var msg [msg_max_len]byte

	for {
		if _, ok := <-intChan; ok {
			for {
				if server_buffer.Len() == 0 {
					break
				}
				sendnode, _ := server_buffer.Front().Value.([]byte)
				DoEncap(msg[:], sendnode[:], server_header[:])
				_, err := dst.Write(msg[0 : header_len+len(sendnode)])
				//fmt.Printf("side 1 mode 0 write[%d]\n", header_len+len(sendnode))
				server_buffer.Remove(server_buffer.Front())
				if err != nil {
					return
				}
				time.Sleep(0)
			}
		} else {
			return
		}
	}
}

func ClientSide(client net.Conn, server net.Conn) {

	go clientCopy(server, client, ENCAP)
	clientCopy(client, server, DECAP)
}

func ServerSide(client net.Conn, server net.Conn) {
	intChan := make(chan int)
	server_buffer := list.New()
	term := new(bool)
	*term = false
	go keep_alive(intChan, term)
	go do_sendmsg(client, server_buffer, intChan)
	go serverCopy(client, server, ENCAP, intChan, server_buffer)
	serverCopy(server, client, DECAP, intChan, server_buffer)
	*term = true
	if len(intChan) != 0 {
		<-intChan
	}
	close(intChan)
}
