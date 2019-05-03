package ws

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/gorilla/websocket"
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

func NewServer(options *Settings) *Server {
	return &Server{
		conns:       make(map[string]*Conn),
		Options:     options,
		connWorkers: &sync.WaitGroup{},
	}
}

type Server struct {
	Options *Settings

	conns       map[string]*Conn
	connMutex   sync.Mutex
	connWorkers *sync.WaitGroup

	isShutDown    bool
	shutdownMutex sync.Mutex
}

func (u *Server) Start(baseCtx context.Context, work *sync.WaitGroup) {
	work.Add(1)
	go u.shutdown(baseCtx, work)
}

func (u *Server) Upgrade(rw http.ResponseWriter, req *http.Request, header http.Header, done <-chan struct{}) (*Conn, error) {
	u.shutdownMutex.Lock()
	defer u.shutdownMutex.Unlock()
	if u.isShutDown {
		return nil, fmt.Errorf("upgrader is going to shutdown and does not accept new connections")
	}

	if header == nil {
		header = http.Header{}
	}
	header.Add("max_ping_interval", u.Options.PingTimeout.String())

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
	for {
		select {
		case <-done:
			u.removeConn(c.ID)
			return
		default:
		}
	}
}

func (u *Server) shutdown(ctx context.Context, work *sync.WaitGroup) {
	<-ctx.Done()
	defer work.Done()

	u.shutdownMutex.Lock()
	u.isShutDown = true
	u.shutdownMutex.Unlock()

	for _, conn := range u.conns {
		close(conn.Shutdown)
	}
	u.connMutex.Lock()
	u.conns = nil
	u.connMutex.Unlock()

	u.connWorkers.Wait()
}

func (u *Server) setCloseHandler(c *Conn) {
	c.SetCloseHandler(func(code int, text string) error {
		u.removeConn(c.ID)
		c.Close()
		return nil
	})
}

func (u *Server) setConnTimeout(c *Conn) {
	c.SetReadDeadline(time.Now().Add(u.Options.PingTimeout))

	c.SetPingHandler(func(message string) error {
		c.SetReadDeadline(time.Now().Add(u.Options.PingTimeout))

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
