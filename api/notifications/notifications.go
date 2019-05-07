package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/ws"
)

const (
	lastKnownRevisionHeader     = "last_known_revision"
	lastKnownRevisionQueryParam = "last_known_revision"
)

func (c *Controller) handleWS(req *web.Request) (*web.Response, error) {
	user, _ := web.UserFromContext(req.Context())
	notificationQueue, lastKnownRevision, err := c.notificator.RegisterConsumer(user)
	if err != nil {
		return nil, fmt.Errorf("could not register notification consumer: %v", err)
	}

	revisionKnownToProxy := 0
	revisionKnownToProxyStr := req.URL.Query().Get(lastKnownRevisionQueryParam)
	if revisionKnownToProxyStr != "" {
		revisionKnownToProxy, err = strconv.Atoi(revisionKnownToProxyStr)
		if err != nil {
			c.unregisterConsumer(notificationQueue)

			log.C(req.Context()).Errorf("could not convert string to number: %v", err)
			return nil, &util.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Description: fmt.Sprintf("invalid %s query parameter", lastKnownRevisionQueryParam),
				ErrorType:   "BadRequest",
			}
		}
	}

	if revisionKnownToProxy > 0 {
		resp, err := c.isRevisionFound(req.Context(), revisionKnownToProxyStr)
		if resp != nil || err != nil {
			return resp, err
		}
	}

	notificationsList, err := c.getNotificationList(req.Context(), user, revisionKnownToProxy, lastKnownRevision)
	if err != nil {
		return nil, err
	}

	rw := req.HijackResponseWriter()

	done := make(chan struct{}, 1)
	conn, err := c.wsServer.Upgrade(rw, req.Request, http.Header{
		lastKnownRevisionHeader: []string{strconv.FormatInt(lastKnownRevision, 10)},
	}, done)
	if err != nil {
		c.unregisterConsumer(notificationQueue)
		return nil, err
	}

	go c.writeLoop(conn, notificationsList, notificationQueue, done)
	go c.readLoop(conn, done)

	return &web.Response{}, nil
}

func (c *Controller) writeLoop(conn *ws.Conn, notificationsList types.ObjectList, q storage.NotificationQueue, done chan<- struct{}) {
	defer c.unregisterConsumer(q)
	defer func() {
		done <- struct{}{}
		conn.Close()
	}()

	for i := 0; i < notificationsList.Len(); i++ {
		notification := (notificationsList.ItemAt(i)).(*types.Notification)
		select {
		case <-conn.Shutdown:
			c.sendWsClose(conn)
			return
		default:
		}

		if !c.sendWsMessage(conn, notification) {
			return
		}
	}

	notificationChannel := q.Channel()

	for {
		select {
		case <-conn.Shutdown:
			c.sendWsClose(conn)
			return
		case notification, ok := <-notificationChannel:
			if !ok {
				c.sendWsClose(conn)
				return
			}

			if !c.sendWsMessage(conn, notification) {
				return
			}
		}
	}
}

func (c *Controller) readLoop(conn *ws.Conn, done chan<- struct{}) {
	defer func() {
		done <- struct{}{}
		conn.Close()
	}()

	for {
		// ReadMessage is needed only to receive ping/pong/close control messages
		// currently we don't expect to receive something else from the proxies
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.D().Errorf("ws: could not read: %v", err)
			return
		}
	}
}

func (c *Controller) sendWsMessage(conn *ws.Conn, msg interface{}) bool {
	conn.SetWriteDeadline(time.Now().Add(c.wsServer.Options.WriteTimeout))
	if err := conn.WriteJSON(msg); err != nil {
		log.D().Errorf("ws: could not write: %v", err)
		return false
	}
	return true
}

func (c *Controller) sendWsClose(conn *ws.Conn) {
	writeDeadline := time.Now().Add(c.wsServer.Options.WriteTimeout)
	if err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""), writeDeadline); err != nil {
		log.D().Errorf("Could not send close message: %v", err)
	}
}

func (c *Controller) unregisterConsumer(q storage.NotificationQueue) {
	if unregErr := c.notificator.UnregisterConsumer(q); unregErr != nil {
		log.D().Errorf("Could not unregister notification consumer: %v", unregErr)
	}
}

func (c *Controller) getNotificationList(ctx context.Context, user *web.UserContext, revisionKnownToProxy int, lastKnownRevision int64) (types.ObjectList, error) {
	listQuery1 := query.ByField(query.GreaterThanOperator, "revision", strconv.Itoa(revisionKnownToProxy))
	// TODO: is this +1 ok or we should add less than or equal operator
	listQuery2 := query.ByField(query.LessThanOperator, "revision", strconv.FormatInt(lastKnownRevision+1, 10))

	platform, err := extractPlatformFromContext(user)
	if err != nil {
		return nil, err
	}
	filterByPlatform := query.ByField(query.EqualsOperator, "platform_id", platform.ID, "")
	notificationsList, err := c.repository.List(ctx, types.NotificationType, listQuery1, listQuery2, filterByPlatform)
	if err != nil {
		// TODO: Wrap err
		return nil, err
	}
	return notificationsList, nil
}

func (c *Controller) isRevisionFound(ctx context.Context, revisionKnownToProxyStr string) (*web.Response, error) {
	listQuery := query.ByField(query.EqualsOperator, "revision", revisionKnownToProxyStr)
	notificationsList, err := c.repository.List(ctx, types.NotificationType, listQuery)
	if err != nil {
		return nil, err
	}
	if notificationsList.Len() != 1 {
		log.C(ctx).Errorf("expected 1 notification, but found: %d", notificationsList.Len())
		return util.NewJSONResponse(http.StatusGone, nil)
	}
	return nil, nil
}

func extractPlatformFromContext(userContext *web.UserContext) (*types.Platform, error) {
	platform := &types.Platform{}
	err := userContext.Data.Data(platform)
	if err != nil {
		return nil, fmt.Errorf("could not get platform from user context %v", err)
	}
	if platform.ID == "" {
		return nil, errors.New("platform ID not found in user context")
	}
	return platform, nil
}
