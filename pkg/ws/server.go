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
func NewServer(baseCtx context.Context, options *Settings) *Server {
	return &Server{
		baseCtx:     baseCtx,
		Options:     options,
		conns:       make(map[string]*Conn),
		connWorkers: &sync.WaitGroup{},
	}
}

// Server is a web socket server which handles connections
type Server struct {
	Options *Settings

	baseCtx     context.Context
	conns       map[string]*Conn
	connMutex   sync.Mutex
	connWorkers *sync.WaitGroup
}

// Start allows server to accept connections and handles shutdown logic
func (s *Server) Start(baseCtx context.Context, work *sync.WaitGroup) {
	work.Add(1)
	go s.shutdown(baseCtx, work)
}

// Upgrade creates the actual web socket connection and handles close events and ping/pong messages.
// Writing to done channel closes the connection.
// If the server context is done, the server will not accept new connections
func (s *Server) Upgrade(rw http.ResponseWriter, req *http.Request, header http.Header, done <-chan struct{}) (*Conn, error) {
	if err := s.baseCtx.Err(); err != nil {
		return nil, fmt.Errorf("upgrader is going to shutdown and does not accept new connections")
	}

	if header == nil {
		header = http.Header{}
	}
	header.Add(maxPingIntervalHeader, s.Options.PingTimeout.String())

	upgrader := &websocket.Upgrader{}
	conn, err := upgrader.Upgrade(rw, req, header)
	if err != nil {
		return nil, err
	}
	wsConn, err := s.addConn(s.baseCtx, conn, s.connWorkers)
	if err != nil {
		return nil, err
	}
	s.setConnTimeout(wsConn)
	s.setCloseHandler(wsConn)

	s.connWorkers.Add(1)
	go s.handleConn(wsConn, done)

	return wsConn, nil
}

func (s *Server) handleConn(c *Conn, done <-chan struct{}) {
	defer s.connWorkers.Done()
	<-done

	if err := s.sendClose(c, websocket.CloseGoingAway); err != nil {
		log.D().Errorf("Could not send close: %v", err)
	}

	if err := c.Close(); err != nil {
		log.D().Errorf("Could not close websocket connection: %v", err)
	}

	s.removeConn(c.ID)
}

func (s *Server) shutdown(ctx context.Context, work *sync.WaitGroup) {
	<-ctx.Done()
	defer work.Done()

	s.connWorkers.Wait()
}

func (s *Server) setCloseHandler(c *Conn) {
	c.SetCloseHandler(func(code int, text string) error {
		log.D().Infof("Websocket received close: %s", text)
		return s.sendClose(c, code)
	})
}

func (s *Server) sendClose(c *Conn, closeCode int) error {
	message := websocket.FormatCloseMessage(closeCode, "")
	err := c.WriteControl(websocket.CloseMessage, message, time.Now().Add(s.Options.WriteTimeout))
	if err != nil && err != websocket.ErrCloseSent {
		log.D().Errorf("Could not write websocket close message: %v", err)
		return err
	}
	return nil
}

func (s *Server) setConnTimeout(c *Conn) {
	if err := c.SetReadDeadline(time.Now().Add(s.Options.PingTimeout)); err != nil {
		log.D().Errorf("Could not set read deadline: %v", err)
	}

	c.SetPingHandler(func(message string) error {
		if err := c.SetReadDeadline(time.Now().Add(s.Options.PingTimeout)); err != nil {
			return err
		}

		err := c.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(s.Options.WriteTimeout))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	})
}

func (s *Server) addConn(baseCtx context.Context, c *websocket.Conn, workGroup *sync.WaitGroup) (*Conn, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		Conn:     c,
		ID:       uuid.String(),
		Shutdown: baseCtx.Done(),
		work:     workGroup,
	}

	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	s.conns[conn.ID] = conn
	return conn, nil
}

func (s *Server) removeConn(id string) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	delete(s.conns, id)
}
