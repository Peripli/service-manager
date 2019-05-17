package notifications

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gorilla/websocket"
)

const (
	MaxPingPeriodHeader = "max_ping_period"
)

func (c *Controller) upgrade(rw http.ResponseWriter, req *http.Request, header http.Header) (*websocket.Conn, error) {
	if header == nil {
		header = http.Header{}
	}
	header.Add(MaxPingPeriodHeader, c.wsSettings.PingTimeout.String())

	upgrader := &websocket.Upgrader{
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			httpErr := &util.HTTPError{
				StatusCode:  status,
				ErrorType:   "WebsocketUpgradeError",
				Description: reason.Error(),
			}
			util.WriteError(httpErr, w)
		},
	}
	conn, err := upgrader.Upgrade(rw, req, header)
	if err != nil {
		return nil, err
	}
	c.configureConn(req.Context(), conn)

	return conn, nil
}

func (c *Controller) configureConn(ctx context.Context, conn *websocket.Conn) {
	if err := conn.SetReadDeadline(time.Now().Add(c.wsSettings.PingTimeout)); err != nil {
		log.C(ctx).WithError(err).Error("Could not set read deadline")
	}

	conn.SetPingHandler(func(message string) error {
		if err := conn.SetReadDeadline(time.Now().Add(c.wsSettings.PingTimeout)); err != nil {
			return err
		}

		err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(c.wsSettings.WriteTimeout))
		if err != nil {
			log.C(ctx).WithError(err).Error("Could not send pong message")
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Temporary() {
				return nil
			}
		}
		return err
	})
}

func (c *Controller) closeConn(ctx context.Context, conn *websocket.Conn, done <-chan struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.C(ctx).Errorf("recovered from panic while closing websocket connection: %s", err)
		}
	}()

	// if base context is cancelled, write loop will quit and write to done
	<-done

	if err := c.sendClose(ctx, conn, websocket.CloseGoingAway); err != nil {
		log.C(ctx).WithError(err).Error("Could not send close")
	}

	if err := conn.Close(); err != nil {
		log.C(ctx).WithError(err).Error("Could not close websocket connection")
	}
}

func (c *Controller) sendClose(ctx context.Context, conn *websocket.Conn, closeCode int) error {
	message := websocket.FormatCloseMessage(closeCode, "")
	err := conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(c.wsSettings.WriteTimeout))
	if err != nil && err != websocket.ErrCloseSent {
		log.C(ctx).WithError(err).Error("Could not write websocket close message")
		return err
	}
	return nil
}
