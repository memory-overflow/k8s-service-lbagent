package common

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/memory-overflow/highly-balanced-scheduling-agent/common/config"
)

// ExternalIP 获取服务ip
func ExternalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("connected to the network?")
}

func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}

func GetDns(ctx context.Context, host string) (ips []string, err error) {
	pwdCmd := exec.Command("nslookup", host)
	pwdOutput, _ := pwdCmd.Output()
	lines := strings.Split(string(pwdOutput), "\n")
	for i := 0; i < len(lines); i++ {
		info := strings.Split(lines[i], ":")
		if len(info) == 2 {
			if strings.TrimSpace(info[0]) == "Name" && strings.TrimSpace(info[1]) == host {
				i++
				if i < len(lines) {
					datas := strings.Split(lines[i], ":")
					if len(datas) == 2 {
						ips = append(ips, strings.TrimSpace(datas[1]))
					}
				}
			}
		}
	}
	if len(ips) == 0 {
		config.GetLogger().Sugar().Errorf("get dns output error: %s, host: %s", string(pwdOutput), host)
		return nil, fmt.Errorf("nslookup %s error: %s", host, string(pwdOutput))
	}
	return ips, nil
}
