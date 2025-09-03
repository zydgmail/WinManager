package guac

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	SocketTimeout  = 15 * time.Second
	MaxGuacMessage = 8192
)

var InternalOpcodeIns = []byte("internal")

// NewGuacamoleTunnel 创建新的Guacamole隧道
func NewGuacamoleTunnel(guacadAddr, protocol, host, port, user, password, uuid string, w, h, dpi int) (s *SimpleTunnel, err error) {
	config := NewGuacamoleConfiguration()
	config.ConnectionID = uuid
	config.Protocol = protocol
	config.OptimalScreenHeight = h
	config.OptimalScreenWidth = w
	config.OptimalResolution = dpi
	config.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}
	config.Parameters = map[string]string{
		"scheme":      protocol,
		"hostname":    host,
		"port":        port,
		"ignore-cert": "true",
		"security":    "",
		"username":    user,
		"password":    password,
	}

	addr, err := net.ResolveTCPAddr("tcp", guacadAddr)
	if err != nil {
		log.Errorf("解析Guacamole地址失败: %v", err)
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Errorf("连接到Guacamole失败: %v", err)
		return nil, err
	}

	stream := NewStream(conn, SocketTimeout)
	// 初始化 rdp/vnc guacd 并认证资产的身份
	err = stream.Handshake(config)
	if err != nil {
		log.Errorf("Guacamole握手失败: %v", err)
		return nil, err
	}

	tunnel := NewSimpleTunnel(stream)
	return tunnel, nil
}
