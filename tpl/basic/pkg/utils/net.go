/**
 * @Author: spruce
 * @Date: 2024-03-28 16:56
 * @Desc: 网络相关函数
 */

package utils

import (
	"errors"
	"net"

	"github.com/gin-gonic/gin"
)

// 获取有效的端口
func AvailablePort() int64 {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	return int64(port)
}

func IPv42Int64(ip string) int64 {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return 0
	}
	b := parsed.To4()
	if b == nil {
		return 0
	}
	return int64(b[3]) | int64(b[2])<<8 | int64(b[1])<<16 | int64(b[0])<<24
}

// 获得请求IP
func ExternalIP() (string, error) {
	iFaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iFace := range iFaces {
		if iFace.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iFace.Flags&net.FlagLoopback != 0 {
			continue // FlagLoopback interface
		}
		addrs, err := iFace.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

// 获得本机IP
func LocalIP() (string, error) {
	addr, err := net.ResolveUDPAddr("udp", "1.2.3.4:1")
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "", err
	}

	defer func(conn *net.UDPConn) {
		_ = conn.Close()
	}(conn)

	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "", err
	}

	return host, nil
}

// 客户端ip
// 直接使用 gin 的 c.ClientIP():它会根据 SetTrustedProxies 配置校验 X-Forwarded-For/X-Real-IP,
// 避免直接信任可伪造的头。注意:使用反向代理时需正确设置可信代理,否则可能拿到伪造 IP。
func ClientIP(c *gin.Context) string {
	return c.ClientIP()
}

// 是否为ipv4地址
func IsIPv4(input string) bool {
	ip := net.ParseIP(input)
	return ip != nil && ip.To4() != nil
}

// 是否为ipv6地址
func IsIPv6(input string) bool {
	ip := net.ParseIP(input)
	return ip != nil && ip.To4() == nil
}
