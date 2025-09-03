package guac

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// Stream wraps the connection to Guacamole providing timeouts and reading
// a single instruction at a time
type Stream struct {
	conn         net.Conn
	ConnectionID string
	parseStart   int
	buffer       []byte
	reset        []byte
	timeout      time.Duration
}

// NewStream creates a new stream
func NewStream(conn net.Conn, timeout time.Duration) *Stream {
	buffer := make([]byte, 0, MaxGuacMessage*3)
	return &Stream{
		conn:    conn,
		timeout: timeout,
		buffer:  buffer,
		reset:   buffer[:cap(buffer)],
	}
}

// Write sends messages to Guacamole with a timeout
func (s *Stream) Write(data []byte) (n int, err error) {
	if err = s.conn.SetWriteDeadline(time.Now().Add(s.timeout)); err != nil {
		log.Errorf("设置写入超时失败: %v", err)
		return
	}
	return s.conn.Write(data)
}

// Available returns true if there are messages buffered
func (s *Stream) Available() bool {
	return len(s.buffer) > 0
}

// Flush resets the internal buffer
func (s *Stream) Flush() {
	copy(s.reset, s.buffer)
	s.buffer = s.reset[:len(s.buffer)]
}

// ReadSome reads a single instruction from the stream
func (s *Stream) ReadSome() (instruction []byte, err error) {
	for {
		// 尝试解析缓冲区中的指令
		if len(s.buffer) > s.parseStart {
			instruction, err = s.parseInstruction()
			if err != nil {
				return nil, err
			}
			if instruction != nil {
				return instruction, nil
			}
		}

		// 需要读取更多数据
		if cap(s.buffer) < MaxGuacMessage {
			s.Flush()
		}

		n, err := s.conn.Read(s.buffer[len(s.buffer):cap(s.buffer)])
		if err != nil && n == 0 {
			switch err.(type) {
			case net.Error:
				ex := err.(net.Error)
				if ex.Timeout() {
					err = ErrUpstreamTimeout.NewError("连接到guacd超时", err.Error())
				} else {
					err = ErrConnectionClosed.NewError("连接到guacd已关闭", err.Error())
				}
			default:
				err = ErrServer.NewError(err.Error())
			}
			return nil, err
		}
		if n == 0 {
			err = ErrServer.NewError("读取0字节")
			return nil, err
		}
		// 必须重新切片以改变长度
		s.buffer = s.buffer[:len(s.buffer)+n]
	}
}

// parseInstruction 解析指令
func (s *Stream) parseInstruction() ([]byte, error) {
	// 简化的指令解析逻辑
	// 在实际实现中，这里应该解析Guacamole协议格式
	if len(s.buffer) > s.parseStart {
		// 查找指令结束符
		for i := s.parseStart; i < len(s.buffer); i++ {
			if s.buffer[i] == ';' {
				instruction := s.buffer[s.parseStart : i+1]
				s.parseStart = i + 1
				return instruction, nil
			}
		}
	}
	return nil, nil
}

// Close closes the underlying network connection
func (s *Stream) Close() error {
	return s.conn.Close()
}

// Handshake configures the guacd session
func (s *Stream) Handshake(config *Config) error {
	// Get protocol / connection ID
	selectArg := config.ConnectionID
	if len(selectArg) == 0 {
		selectArg = config.Protocol
	}

	// Send requested protocol or connection ID
	_, err := s.Write(NewInstruction("select", selectArg).Byte())
	if err != nil {
		return err
	}

	// Wait for server Args
	args, err := s.AssertOpcode("args")
	if err != nil {
		return err
	}

	// Build Args list off provided names and config
	argNameS := args.Args
	argValueS := make([]string, 0, len(argNameS))
	for _, argName := range argNameS {
		// Get defined value for name
		value := config.Parameters[argName]
		// If value defined, set that value
		if len(value) == 0 {
			value = ""
		}
		argValueS = append(argValueS, value)
	}

	// Send size
	_, err = s.Write(NewInstruction("size",
		fmt.Sprintf("%v", config.OptimalScreenWidth),
		fmt.Sprintf("%v", config.OptimalScreenHeight),
		fmt.Sprintf("%v", config.OptimalResolution)).Byte(),
	)

	if err != nil {
		return err
	}

	// Send supported audio formats
	_, err = s.Write(NewInstruction("audio", config.AudioMimetypes...).Byte())
	if err != nil {
		return err
	}

	// Send supported video formats
	_, err = s.Write(NewInstruction("video", config.VideoMimetypes...).Byte())
	if err != nil {
		return err
	}

	// Send supported image formats
	_, err = s.Write(NewInstruction("image", config.ImageMimetypes...).Byte())
	if err != nil {
		return err
	}

	// Send Args
	_, err = s.Write(NewInstruction("connect", argValueS...).Byte())
	if err != nil {
		return err
	}

	// Wait for ready, store ID
	ready, err := s.AssertOpcode("ready")
	if err != nil {
		return err
	}

	readyArgs := ready.Args
	if len(readyArgs) == 0 {
		err = ErrServer.NewError("未收到连接ID")
		return err
	}

	s.Flush()
	s.ConnectionID = readyArgs[0]

	return nil
}

// AssertOpcode 断言操作码
func (s *Stream) AssertOpcode(opcode string) (*Instruction, error) {
	instruction, err := s.ReadSome()
	if err != nil {
		return nil, err
	}

	parsed, err := Parse(instruction)
	if err != nil {
		return nil, err
	}

	if parsed.Opcode != opcode {
		return nil, ErrServer.NewError("期望操作码", opcode, "但收到", parsed.Opcode)
	}

	return parsed, nil
}
