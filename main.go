package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

func main() {
	var mode []byte
	var listen_addr []byte
	var remote_addr []byte
	data, _ := ioutil.ReadFile("./socks.conf")

	lines := strings.Split(string(data), "\n")
	for idx, aline := range lines {
		if idx > 2 {
			break
		}
		pos := strings.Index(aline, "=")
		key := aline[0 : pos-1]

		ret := strings.Contains(key, "mode")
		if ret == true {
			if aline[len(aline)-1] == '\r' {
				mode = []byte(aline[pos+2 : len(aline)-1])
			} else {
				mode = []byte(aline[pos+2 : len(aline)])
			}
			continue
		}
		ret = strings.Contains(key, "listen_addr")
		if ret == true {
			if aline[len(aline)-1] == '\r' {
				listen_addr = []byte(aline[pos+2 : len(aline)-1])
			} else {
				listen_addr = []byte(aline[pos+2 : len(aline)])
			}
			continue
		}
		ret = strings.Contains(key, "remote_addr")
		if ret == true {
			if aline[len(aline)-1] == '\r' {
				remote_addr = []byte(aline[pos+2 : len(aline)-1])
			} else {
				remote_addr = []byte(aline[pos+2 : len(aline)])
			}
			continue
		}
	}
	fmt.Printf("mode %s %d\n", string(mode[:]), len(mode))
	fmt.Printf("listen_addr %s %d\n", string(listen_addr[:]), len(listen_addr))
	fmt.Printf("remote_addr %s %d\n", string(remote_addr[:]), len(remote_addr))
	ret := strings.Compare(string(mode[:]), "test")
	if ret == 0 {
		go ServerDaemon(string(remote_addr[:]))
		ClientDaemon(string(listen_addr[:]), string(remote_addr[:]))
	}

	ret = strings.Compare(string(mode[:]), "client")
	if ret == 0 {
		ClientDaemon(string(listen_addr[:]), string(remote_addr[:]))
	}
	ret = strings.Compare(string(mode[:]), "server")
	if ret == 0 {
		ServerDaemon(string(listen_addr[:]))
	}
}
