package ws

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/gofrs/uuid"

	"github.com/gorilla/websocket"
)

const (
	maxPingIntervalHeader = "max_ping_interval"
)

type Settings struct {
	PingTimeout  time.Duration `mapstructure:"ping_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DefaultSettings return the default values for ws server
func DefaultSettings() *Settings {
	return &Settings{
		PingTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}
}

// Validate validates the ws server settings
func (s *Settings) Validate() error {
	if s.PingTimeout <= 0 {
		return fmt.Errorf("validate Settings: PingTimeut should be > 0")
	}

	if s.WriteTimeout <= 0 {
		return fmt.Errorf("validate Settings: WriteTimeout should be > 0")
	}

	return nil
}

// NewServer create new websocket server
func NewServer(options *Settings) *Server {
	return &Server{
		conns:       make(map[string]*Conn),
		Options:     options,
		connWorkers: &sync.WaitGroup{},
	}
}

// Server is a web socket server which handles connections
type Server struct {
	Options *Settings

	conns       map[string]*Conn
	connMutex   sync.Mutex
	connWorkers *sync.WaitGroup

	isShutDown    bool
	shutdownMutex sync.RWMutex
}

// Start allows server to accept connections and handles shutdown logic
func (u *Server) Start(baseCtx context.Context, work *sync.WaitGroup) {
	work.Add(1)
	go u.shutdown(baseCtx, work)
}

// Upgrade creates the actual web socket connection and handles close events and ping/pong messages.
// Writing to done channel closes the connection.
// If the server context is done, the server will not accept new connections
func (u *Server) Upgrade(rw http.ResponseWriter, req *http.Request, header http.Header, done <-chan struct{}) (*Conn, error) {
	u.shutdownMutex.RLock()
	defer u.shutdownMutex.RUnlock()
	if u.isShutDown {
		return nil, fmt.Errorf("upgrader is going to shutdown and does not accept new connections")
	}

	if header == nil {
		header = http.Header{}
	}
	header.Add(maxPingIntervalHeader, u.Options.PingTimeout.String())

	upgrader := &websocket.Upgrader{}
	conn, err := upgrader.Upgrade(rw, req, header)
	if err != nil {
		return nil, err
	}
	wsConn, err := u.addConn(conn, u.connWorkers)
	if err != nil {
		return nil, err
	}
	u.setConnTimeout(wsConn)
	u.setCloseHandler(wsConn)

	u.connWorkers.Add(1)
	go u.handleConn(wsConn, done)

	return wsConn, nil
}

func (u *Server) handleConn(c *Conn, done <-chan struct{}) {
	defer u.connWorkers.Done()
	<-done

	if err := u.sendClose(c, websocket.CloseGoingAway); err != nil {
		log.D().Errorf("Could not send close: %v", err)
	}

	if err := c.Close(); err != nil {
		log.D().Errorf("Could not close websocket connection: %v", err)
	}

	u.removeConn(c.ID)
}

func (u *Server) shutdown(ctx context.Context, work *sync.WaitGroup) {
	<-ctx.Done()
	defer work.Done()

	u.shutdownMutex.Lock()
	u.isShutDown = true
	u.shutdownMutex.Unlock()

	func() {
		u.connMutex.Lock()
		defer u.connMutex.Unlock()

		for _, conn := range u.conns {
			close(conn.Shutdown)
		}
	}()

	u.connWorkers.Wait()
}

func (u *Server) setCloseHandler(c *Conn) {
	c.SetCloseHandler(func(code int, text string) error {
		log.D().Infof("Websocket received close: %s", text)
		return u.sendClose(c, code)
	})
}

func (u *Server) sendClose(c *Conn, closeCode int) error {
	message := websocket.FormatCloseMessage(closeCode, "")
	err := c.WriteControl(websocket.CloseMessage, message, time.Now().Add(u.Options.WriteTimeout))
	if err != nil && err != websocket.ErrCloseSent {
		log.D().Errorf("Could not write websocket close message: %v", err)
		return err
	}
	return nil
}

func (u *Server) setConnTimeout(c *Conn) {
	if err := c.SetReadDeadline(time.Now().Add(u.Options.PingTimeout)); err != nil {
		log.D().Errorf("Could not set read deadline: %v", err)
	}

	c.SetPingHandler(func(message string) error {
		if err := c.SetReadDeadline(time.Now().Add(u.Options.PingTimeout)); err != nil {
			return err
		}

		err := c.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(u.Options.WriteTimeout))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	})
}

func (u *Server) addConn(c *websocket.Conn, workGroup *sync.WaitGroup) (*Conn, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		Conn:     c,
		ID:       uuid.String(),
		Shutdown: make(chan struct{}),
		work:     workGroup,
	}

	u.connMutex.Lock()
	defer u.connMutex.Unlock()
	u.conns[conn.ID] = conn
	return conn, nil
}

func (u *Server) removeConn(id string) {
	u.connMutex.Lock()
	defer u.connMutex.Unlock()
	delete(u.conns, id)
}
