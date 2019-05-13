package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/gorilla/websocket"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
)

const (
	LastKnownRevisionHeader     = "last_notification_revision"
	LastKnownRevisionQueryParam = "last_notification_revision"
)

var errRevisionNotFound = errors.New("revision not found")

func (c *Controller) handleWS(req *web.Request) (*web.Response, error) {
	ctx := req.Context()
	logger := log.C(ctx)

	user, ok := web.UserFromContext(req.Context())
	if !ok {
		return nil, errors.New("user details not found in request context")
	}

	revisionKnownToProxy := int64(-1)
	revisionKnownToProxyStr := req.URL.Query().Get(LastKnownRevisionQueryParam)
	if revisionKnownToProxyStr != "" {
		var err error
		revisionKnownToProxy, err = strconv.ParseInt(revisionKnownToProxyStr, 10, 64)
		if err != nil {
			logger.Errorf("could not convert string %s to number: %v", revisionKnownToProxyStr, err)
			return nil, &util.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Description: fmt.Sprintf("invalid %s query parameter", LastKnownRevisionQueryParam),
				ErrorType:   "BadRequest",
			}
		}
	}

	if revisionKnownToProxy == 0 {
		return util.NewJSONResponse(http.StatusGone, nil)
	}

	notificationQueue, lastKnownRevision, notificationsList, err := c.registerConsumer(ctx, revisionKnownToProxy, user)
	if err != nil {
		if err == errRevisionNotFound {
			return util.NewJSONResponse(http.StatusGone, nil)
		}
		return nil, err
	}

	correlationID := logger.Data[log.FieldCorrelationID].(string)
	childCtx := newContextWithCorrelationID(c.baseCtx, correlationID)

	defer func() {
		if err := recover(); err != nil {
			log.C(childCtx).Errorf("recovered from panic while establishing websocket connection: %s", err)
		}
	}()

	rw := req.HijackResponseWriter()
	responseHeaders := http.Header{
		LastKnownRevisionHeader: []string{strconv.FormatInt(lastKnownRevision, 10)},
	}

	conn, err := c.upgrade(rw, req.Request, responseHeaders)
	if err != nil {
		c.unregisterConsumer(ctx, notificationQueue)
		return nil, err
	}

	done := make(chan struct{}, 2)

	go c.closeConn(childCtx, conn, done)
	go c.writeLoop(childCtx, conn, notificationsList, notificationQueue, done)
	go c.readLoop(childCtx, conn, done)

	return &web.Response{}, nil
}

func (c *Controller) writeLoop(ctx context.Context, conn *websocket.Conn, notificationsList *types.Notifications, q storage.NotificationQueue, done chan<- struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.C(ctx).Errorf("recovered from panic while writing to websocket connection: %s", err)
		}
	}()

	defer func() {
		done <- struct{}{}
	}()
	defer c.unregisterConsumer(ctx, q)

	for _, notification := range notificationsList.Notifications {
		select {
		case <-ctx.Done():
			log.C(ctx).Infof("Websocket connection shutting down")
			return
		default:
		}

		if !c.sendWsMessage(ctx, conn, notification) {
			return
		}
	}

	notificationChannel := q.Channel()

	for {
		select {
		case <-ctx.Done():
			log.C(ctx).Infof("Websocket connection shutting down")
			return
		case notification, ok := <-notificationChannel:
			if !ok {
				log.C(ctx).Infof("Notifications channel is closed. Closing websocket connection...")
				return
			}

			if !c.sendWsMessage(ctx, conn, notification) {
				return
			}
		}
	}
}

func (c *Controller) readLoop(ctx context.Context, conn *websocket.Conn, done chan<- struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.C(ctx).Errorf("recovered from panic while reading from websocket connection: %s", err)
		}
	}()

	defer func() {
		done <- struct{}{}
	}()

	for {
		// ReadMessage is needed only to receive ping/pong/close control messages
		// currently we don't expect to receive something else from the proxies
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.C(ctx).Errorf("ws: could not read: %v", err)
			return
		}
	}
}

func (c *Controller) sendWsMessage(ctx context.Context, conn *websocket.Conn, msg interface{}) bool {
	if err := conn.SetWriteDeadline(time.Now().Add(c.wsSettings.WriteTimeout)); err != nil {
		log.C(ctx).Errorf("Could not set write deadline: %v", err)
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.C(ctx).Errorf("ws: could not write: %v", err)
		return false
	}
	return true
}

func (c *Controller) registerConsumer(ctx context.Context, revisionKnownToProxy int64, user *web.UserContext) (storage.NotificationQueue, int64, *types.Notifications, error) {
	var (
		notificationQueue storage.NotificationQueue
		notificationsList *types.Notifications
		lastKnownRevision int64
		err               error
	)
	notificationQueue, lastKnownRevision, err = c.notificator.RegisterConsumer(user)
	if err != nil {
		return nil, -1, nil, fmt.Errorf("could not register notification consumer: %v", err)
	}
	defer func() {
		if err != nil {
			c.unregisterConsumer(ctx, notificationQueue)
		}
	}()

	if lastKnownRevision < revisionKnownToProxy {
		log.C(ctx).Infof("Notification revision known to proxy %d is greater than revision known to SM %d", revisionKnownToProxy, lastKnownRevision)
		return nil, -1, nil, errRevisionNotFound
	}

	notificationsList, err = c.getNotificationList(ctx, user, revisionKnownToProxy, lastKnownRevision)
	if err != nil {
		return nil, -1, nil, err
	}

	if revisionKnownToProxy > 0 {
		var notification *types.Notification
		if notificationsList.Len() > 0 {
			notification = notificationsList.Notifications[0]
			notificationsList.Notifications = notificationsList.Notifications[1:]
		}

		if notification == nil || notification.Revision != revisionKnownToProxy {
			log.C(ctx).Infof("Notification with revision %d known to proxy not found", revisionKnownToProxy)
			return nil, -1, nil, errRevisionNotFound
		}
	}
	return notificationQueue, lastKnownRevision, notificationsList, err
}

func (c *Controller) unregisterConsumer(ctx context.Context, q storage.NotificationQueue) {
	if unregErr := c.notificator.UnregisterConsumer(q); unregErr != nil {
		log.C(ctx).Errorf("Could not unregister notification consumer: %v", unregErr)
	}
}

func (c *Controller) getNotificationList(ctx context.Context, user *web.UserContext, revisionKnownToProxy, LastKnownRevisionHeader int64) (*types.Notifications, error) {
	// TODO: is this +1/-1 ok or we should add less than or equal operator
	listQuery1 := query.ByField(query.GreaterThanOperator, "revision", strconv.FormatInt(revisionKnownToProxy-1, 10))
	listQuery2 := query.ByField(query.LessThanOperator, "revision", strconv.FormatInt(LastKnownRevisionHeader+1, 10))

	platform, err := extractPlatformFromContext(user)
	if err != nil {
		return nil, err
	}
	filterByPlatform := query.ByField(query.EqualsOrNilOperator, "platform_id", platform.ID)
	objectList, err := c.repository.List(ctx, types.NotificationType, listQuery1, listQuery2, filterByPlatform)
	if err != nil {
		return nil, err
	}
	notificationsList := objectList.(*types.Notifications)
	// TODO: Should be done in the database with order by
	sort.Slice(notificationsList.Notifications, func(i, j int) bool {
		return notificationsList.Notifications[i].Revision < notificationsList.Notifications[j].Revision
	})

	return notificationsList, nil
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

func newContextWithCorrelationID(baseCtx context.Context, correlationID string) context.Context {
	entry := log.C(baseCtx).WithField(log.FieldCorrelationID, correlationID)
	return log.ContextWithLogger(baseCtx, entry)
}
