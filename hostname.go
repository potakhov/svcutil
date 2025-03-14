package svcutil

import (
	"net"
	"os"
	"strings"
)

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = GetLocalIP()
	}

	hostname = strings.Replace(hostname, "-", "_", -1)
	hostname = strings.Replace(hostname, ".", "_", -1)
	hostname = strings.Replace(hostname, "*", "_", -1)
	hostname = strings.Replace(hostname, ">", "_", -1)

	return hostname
}
