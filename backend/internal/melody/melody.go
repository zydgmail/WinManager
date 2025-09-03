package melody

import (
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type handleMessageFunc func(*Session, []byte)
type handleErrorFunc func(*Session, error)
type handleCloseFunc func(*Session, int, string) error
type handleSessionFunc func(*Session)

// Melody implements a websocket manager.
type Melody struct {
	Config                   *Config
	Upgrader                 *websocket.Upgrader
	messageHandler           handleMessageFunc
	messageHandlerBinary     handleMessageFunc
	messageSentHandler       handleMessageFunc
	messageSentHandlerBinary handleMessageFunc
	errorHandler             handleErrorFunc
	closeHandler             handleCloseFunc
	connectHandler           handleSessionFunc
	disconnectHandler        handleSessionFunc
	pongHandler              handleSessionFunc
	hub                      *hub
}

// New creates a new melody instance with default Upgrader and Config.
func New() *Melody {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	hub := newHub()

	go hub.run()

	return &Melody{
		Config:                   newConfig(),
		Upgrader:                 upgrader,
		messageHandler:           func(*Session, []byte) {},
		messageHandlerBinary:     func(*Session, []byte) {},
		messageSentHandler:       func(*Session, []byte) {},
		messageSentHandlerBinary: func(*Session, []byte) {},
		errorHandler:             func(*Session, error) {},
		closeHandler:             nil,
		connectHandler:           func(*Session) {},
		disconnectHandler:        func(*Session) {},
		pongHandler:              func(*Session) {},
		hub:                      hub,
	}
}

// HandleConnect fires fn when a session connects.
func (m *Melody) HandleConnect(fn func(*Session)) {
	m.connectHandler = fn
}

// HandleDisconnect fires fn when a session disconnects.
func (m *Melody) HandleDisconnect(fn func(*Session)) {
	m.disconnectHandler = fn
}

// HandleMessage fires fn when a text message comes in.
func (m *Melody) HandleMessage(fn func(*Session, []byte)) {
	m.messageHandler = fn
}

// HandleMessageBinary fires fn when a binary message comes in.
func (m *Melody) HandleMessageBinary(fn func(*Session, []byte)) {
	m.messageHandlerBinary = fn
}

// HandleError fires fn when a session has an error.
func (m *Melody) HandleError(fn func(*Session, error)) {
	m.errorHandler = fn
}

// HandleClose sets the handler for close messages received from the session.
func (m *Melody) HandleClose(fn func(*Session, int, string) error) {
	if fn != nil {
		m.closeHandler = fn
	}
}

// HandleRequest upgrades http requests to websocket connections and dispatches them to be handled by the melody instance.
func (m *Melody) HandleRequest(w http.ResponseWriter, r *http.Request, header http.Header) error {
	return m.HandleRequestWithKeys(w, r, header, nil)
}

// HandleRequestWithKeys does the same as HandleRequest but populates session.Keys with keys.
func (m *Melody) HandleRequestWithKeys(w http.ResponseWriter, r *http.Request, header http.Header, keys map[string]interface{}) error {
	if m.hub.closed() {
		return errors.New("melody instance is closed")
	}

	conn, err := m.Upgrader.Upgrade(w, r, header)

	if err != nil {
		return err
	}

	session := &Session{
		Request: r,
		Keys:    keys,
		UUID:    generateUUID(),
		conn:    conn,
		output:  make(chan *envelope, m.Config.MessageBufferSize),
		melody:  m,
		open:    true,
		rwmutex: &sync.RWMutex{},
	}

	m.hub.register <- session

	m.connectHandler(session)

	go session.writePump()

	session.readPump()

	if !m.hub.closed() {
		m.hub.unregister <- session
	}

	session.close()

	m.disconnectHandler(session)

	session.Keys = nil

	return nil
}

// Broadcast broadcasts a text message to all sessions.
func (m *Melody) Broadcast(msg []byte) error {
	if m.hub.closed() {
		return errors.New("melody instance is closed")
	}

	message := &envelope{t: websocket.TextMessage, msg: msg}
	m.hub.queue <- message

	return nil
}

// BroadcastBinary broadcasts a binary message to all sessions.
func (m *Melody) BroadcastBinary(msg []byte) error {
	if m.hub.closed() {
		return errors.New("melody instance is closed")
	}

	message := &envelope{t: websocket.BinaryMessage, msg: msg}
	m.hub.queue <- message

	return nil
}

// Close closes the melody instance and all connected sessions.
func (m *Melody) Close() error {
	if m.hub.closed() {
		return errors.New("melody instance is already closed")
	}

	m.hub.exit <- &envelope{t: websocket.CloseMessage, msg: []byte{}}

	return nil
}

// Len return the number of connected sessions.
func (m *Melody) Len() int {
	return m.hub.len()
}
