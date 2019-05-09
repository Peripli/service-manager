package ws

import (
	"github.com/gorilla/websocket"
)

type Conn struct {
	*websocket.Conn
	ID string

	Shutdown <-chan struct{}
}
