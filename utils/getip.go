package utils

import (
	"net"
	"strings"
)

func GetOutBoundIP() (string, error) {
	conn, err := net.Dial("udp", "8:8:8:8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	// UDPAddr 表示 UDP 端点的地址。
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := strings.Split(localAddr.IP.String(), ":")[0]
	return ip, nil
}
