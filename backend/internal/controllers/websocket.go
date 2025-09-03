package controllers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"winmanager-backend/internal/config"
	"winmanager-backend/internal/guac"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/melody"
	"winmanager-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

// Melody WebSocket管理器实例
var Melody = melody.New()

// ReqArg WebSocket请求参数
type ReqArg struct {
	Width  int `form:"width"`
	Height int `form:"height"`
	DPI    int `form:"dpi"`
}

// WsController WebSocket控制器
func WsController() gin.HandlerFunc {
	websocketReadBufferSize := guac.MaxGuacMessage
	websocketWriteBufferSize := guac.MaxGuacMessage * 2
	upgrade := websocket.Upgrader{
		ReadBufferSize:  websocketReadBufferSize,
		WriteBufferSize: websocketWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c *gin.Context) {
		// /ws/:id
		targetHost := c.Param("id")
		parallelHosts := strings.Split(c.Query("hosts"), ",")

		logger.Infof("WebSocket连接到目标主机: %s, 并行主机: %v", targetHost, parallelHosts)

		arg := new(ReqArg)
		err := c.BindQuery(arg)
		if err != nil {
			logger.Errorf("绑定查询参数失败: %v", err)
			c.JSON(202, err.Error())
			return
		}

		// 设置默认值
		if arg.Width == 0 {
			arg.Width = 1920
		}
		if arg.Height == 0 {
			arg.Height = 1080
		}
		if arg.DPI == 0 {
			arg.DPI = 150
		}

		protocol := c.Request.Header.Get("Sec-Websocket-Protocol")
		ws, err := upgrade.Upgrade(c.Writer, c.Request, http.Header{
			"Sec-Websocket-Protocol": {protocol},
		})
		if err != nil {
			logger.Errorf("升级WebSocket失败: %v", err)
			return
		}
		defer func() {
			if err = ws.Close(); err != nil {
				logger.Errorf("关闭WebSocket连接失败: %v", err)
			}
		}()

		// 创建Guacamole隧道连接
		pipeTunnel, err := guac.NewGuacamoleTunnel("127.0.0.1:4822",
			"vnc", targetHost, "5900", "root", "viu@1234", "",
			arg.Width, arg.Height, arg.DPI)
		if err != nil {
			logger.Errorf("创建Guacamole隧道失败: %v", err)
			return
		}
		defer pipeTunnel.Close()

		logger.Infof("WebSocket连接建立成功: %s", targetHost)

		// 处理WebSocket和Guacamole之间的数据传输
		handleWebSocketTunnel(ws, pipeTunnel)
	}
}

// StateController WebSocket状态控制器
func StateController() gin.HandlerFunc {
	websocketReadBufferSize := guac.MaxGuacMessage
	websocketWriteBufferSize := guac.MaxGuacMessage * 2
	upgrade := websocket.Upgrader{
		ReadBufferSize:  websocketReadBufferSize,
		WriteBufferSize: websocketWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c *gin.Context) {
		// /ws/state/:id
		hosts := []string{c.Param("id")}
		parallelHosts := strings.Split(c.Query("hosts"), ",")
		parallelHosts = append(parallelHosts, c.Param("id"))

		logger.Infof("WebSocket状态连接到主机: %v", parallelHosts)

		arg := new(ReqArg)
		err := c.BindQuery(arg)
		if err != nil {
			logger.Errorf("绑定查询参数失败: %v", err)
			c.JSON(202, err.Error())
			return
		}

		protocol := c.Request.Header.Get("Sec-Websocket-Protocol")
		ws, err := upgrade.Upgrade(c.Writer, c.Request, http.Header{
			"Sec-Websocket-Protocol": {protocol},
		})

		if err != nil {
			logger.Errorf("升级WebSocket失败: %v", err)
			return
		}
		defer func() {
			if err = ws.Close(); err != nil {
				logger.Errorf("关闭WebSocket连接失败: %v", err)
			}
		}()

		// 处理状态同步
		handleStateSync(ws, hosts, parallelHosts)
	}
}

// handleWebSocketTunnel 处理WebSocket和Guacamole隧道之间的数据传输
func handleWebSocketTunnel(ws *websocket.Conn, tunnel *guac.SimpleTunnel) {
	eg, _ := errgroup.WithContext(context.Background())
	reader := tunnel.AcquireReader()
	writer := tunnel.AcquireWriter()

	// 从Guacamole读取数据并发送到WebSocket
	eg.Go(func() error {
		buf := bytes.NewBuffer(make([]byte, 0, guac.MaxGuacMessage*2))

		for {
			ins, err := reader.ReadSome()
			if err != nil {
				return err
			}

			if bytes.HasPrefix(ins, guac.InternalOpcodeIns) {
				// 内部操作码消息不发送到WebSocket
				continue
			}

			if _, err = buf.Write(ins); err != nil {
				return err
			}

			// 如果缓冲区没有更多数据或达到最大缓冲区大小，发送数据并重置
			if !reader.Available() || buf.Len() >= guac.MaxGuacMessage {
				if err = ws.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
					if err == websocket.ErrCloseSent {
						return fmt.Errorf("websocket: %v", err)
					}
					logger.Errorf("发送消息到WebSocket失败: %v", err)
					return err
				}
				buf.Reset()
			}
		}
	})

	// 从WebSocket读取数据并发送到Guacamole
	eg.Go(func() error {
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				logger.Errorf("从WebSocket读取消息失败: %v", err)
				return err
			}

			if _, err = writer.Write(data); err != nil {
				logger.Errorf("写入到Guacamole失败: %v", err)
				return err
			}
		}
	})

	if err := eg.Wait(); err != nil {
		logger.Errorf("WebSocket隧道会话错误: %v", err)
	}
}

// handleStateSync 处理状态同步
func handleStateSync(ws *websocket.Conn, hosts []string, parallelHosts []string) {
	eg, _ := errgroup.WithContext(context.Background())
	currentHosts := parallelHosts

	logger.Infof("开始状态同步: 主机=%v, 并行主机=%v", hosts, parallelHosts)

	// 处理WebSocket消息
	eg.Go(func() error {
		for {
			messageType, data, err := ws.ReadMessage()
			if err != nil {
				logger.Errorf("读取WebSocket消息失败: %v", err)
				return err
			}

			if messageType == websocket.TextMessage {
				message := string(data)
				logger.Debugf("收到状态消息: %s", message)

				// 处理状态同步消息
				if strings.Contains(message, "toggle") {
					// 处理同步切换
					logger.Infof("处理同步切换: %s", message)
					handleSyncToggle(currentHosts, message)
				} else {
					// 处理其他同步命令
					handleSyncCommand(currentHosts, message)
				}
			}
		}
	})

	// 发送心跳
	eg.Go(func() error {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					logger.Errorf("发送心跳失败: %v", err)
					return err
				}
			}
		}
	})

	if err := eg.Wait(); err != nil {
		logger.Errorf("状态同步会话错误: %v", err)
	}
}

// handleSyncToggle 处理同步切换命令
func handleSyncToggle(hosts []string, message string) {
	logger.Infof("执行同步切换: 主机数=%d, 命令=%s", len(hosts), message)

	// 并发执行切换命令
	var wg sync.WaitGroup
	ch := make(chan struct{}, 10) // 限制并发数为10

	for _, host := range hosts {
		if host == "" {
			continue
		}

		ch <- struct{}{}
		wg.Add(1)

		go func(hostLan string) {
			defer wg.Done()
			defer func() { <-ch }()

			// 获取实例信息
			instance, err := models.GetInstanceByLan(hostLan)
			if err != nil {
				logger.Errorf("获取实例失败: LAN=%s, 错误=%v", hostLan, err)
				return
			}

			// 执行切换命令
			agentHTTPPort := config.GetAgentHTTPPort()
			toggleURL := fmt.Sprintf("http://%s:%d/api/toggle", instance.Lan, agentHTTPPort)
			resp, err := http.Get(toggleURL)
			if err != nil {
				logger.Errorf("同步切换失败: %s, 错误=%v", hostLan, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				logger.Infof("同步切换成功: %s", hostLan)
			} else {
				logger.Errorf("同步切换失败: %s, 状态码=%d", hostLan, resp.StatusCode)
			}
		}(host)
	}

	wg.Wait()
	logger.Infof("同步切换完成: 处理了 %d 个主机", len(hosts))
}

// handleSyncCommand 处理其他同步命令
func handleSyncCommand(hosts []string, command string) {
	logger.Infof("执行同步命令: 主机数=%d, 命令=%s", len(hosts), command)

	// 解析命令类型
	var endpoint string
	if strings.Contains(command, "screenshot") {
		endpoint = "/api/screenshot"
	} else if strings.Contains(command, "restart") {
		endpoint = "/api/restart"
	} else if strings.Contains(command, "status") {
		endpoint = "/api/status"
	} else {
		logger.Warnf("未知的同步命令: %s", command)
		return
	}

	// 并发执行命令
	var wg sync.WaitGroup
	ch := make(chan struct{}, 10) // 限制并发数为10

	for _, host := range hosts {
		if host == "" {
			continue
		}

		ch <- struct{}{}
		wg.Add(1)

		go func(hostLan string) {
			defer wg.Done()
			defer func() { <-ch }()

			// 获取实例信息
			instance, err := models.GetInstanceByLan(hostLan)
			if err != nil {
				logger.Errorf("获取实例失败: LAN=%s, 错误=%v", hostLan, err)
				return
			}

			// 执行命令
			agentHTTPPort := config.GetAgentHTTPPort()
			commandURL := fmt.Sprintf("http://%s:%d%s", instance.Lan, agentHTTPPort, endpoint)
			resp, err := http.Get(commandURL)
			if err != nil {
				logger.Errorf("同步命令失败: %s%s, 错误=%v", hostLan, endpoint, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				logger.Infof("同步命令成功: %s%s", hostLan, endpoint)
			} else {
				logger.Errorf("同步命令失败: %s%s, 状态码=%d", hostLan, endpoint, resp.StatusCode)
			}
		}(host)
	}

	wg.Wait()
	logger.Infof("同步命令完成: 处理了 %d 个主机", len(hosts))
}
