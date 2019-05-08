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

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/ws"
)

const (
	lastKnownRevisionHeader     = "last_known_revision"
	lastKnownRevisionQueryParam = "last_known_revision"
)

var errRevisionNotFound error = errors.New("revision not found")

func (c *Controller) handleWS(req *web.Request) (*web.Response, error) {
	user, ok := web.UserFromContext(req.Context())
	if !ok {
		return nil, errors.New("user details not found in request context")
	}
	ctx := req.Context()

	revisionKnownToProxy := int64(0)
	revisionKnownToProxyStr := req.URL.Query().Get(lastKnownRevisionQueryParam)
	if revisionKnownToProxyStr != "" {
		var err error
		revisionKnownToProxy, err = strconv.ParseInt(revisionKnownToProxyStr, 10, 64)
		if err != nil {
			log.C(ctx).Errorf("could not convert string to number: %v", err)
			return nil, &util.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Description: fmt.Sprintf("invalid %s query parameter", lastKnownRevisionQueryParam),
				ErrorType:   "BadRequest",
			}
		}
	}

	notificationQueue, lastKnownRevision, notificationsList, err := c.registerConsumer(ctx, revisionKnownToProxy, user)
	if err == errRevisionNotFound {
		return util.NewJSONResponse(http.StatusGone, nil)
	}

	rw := req.HijackResponseWriter()
	done := make(chan struct{}, 1)
	conn, err := c.wsServer.Upgrade(rw, req.Request, http.Header{
		lastKnownRevisionHeader: []string{strconv.FormatInt(lastKnownRevision, 10)},
	}, done)
	if err != nil {
		c.unregisterConsumer(ctx, notificationQueue)
		return nil, err
	}

	go c.writeLoop(ctx, conn, notificationsList, notificationQueue, done)
	go c.readLoop(ctx, conn, done)

	return &web.Response{}, nil
}

func (c *Controller) writeLoop(ctx context.Context, conn *ws.Conn, notificationsList *types.Notifications, q storage.NotificationQueue, done chan<- struct{}) {
	defer c.unregisterConsumer(ctx, q)
	defer func() {
		done <- struct{}{}
	}()

	for i := 0; i < notificationsList.Len(); i++ {
		notification := (notificationsList.ItemAt(i)).(*types.Notification)
		select {
		case <-conn.Shutdown:
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
		case <-conn.Shutdown:
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

func (c *Controller) readLoop(ctx context.Context, conn *ws.Conn, done chan<- struct{}) {
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

func (c *Controller) sendWsMessage(ctx context.Context, conn *ws.Conn, msg interface{}) bool {
	if err := conn.SetWriteDeadline(time.Now().Add(c.wsServer.Options.WriteTimeout)); err != nil {
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

	if notificationsList.Len() > 0 && revisionKnownToProxy > 0 {
		// TODO: we expect that notificationsList is ordered by revision
		notification := notificationsList.ItemAt(0).(*types.Notification)
		if notification.Revision != revisionKnownToProxy {
			log.C(ctx).Infof("Notification with revision %d known to proxy not found", revisionKnownToProxy)
			return nil, -1, nil, errRevisionNotFound
		}
		notificationsList.Notifications = notificationsList.Notifications[1:]
	}
	return notificationQueue, lastKnownRevision, notificationsList, err
}

func (c *Controller) unregisterConsumer(ctx context.Context, q storage.NotificationQueue) {
	if unregErr := c.notificator.UnregisterConsumer(q); unregErr != nil {
		log.C(ctx).Errorf("Could not unregister notification consumer: %v", unregErr)
	}
}

func (c *Controller) getNotificationList(ctx context.Context, user *web.UserContext, revisionKnownToProxy, lastKnownRevision int64) (*types.Notifications, error) {
	// TODO: is this +1/-1 ok or we should add less than or equal operator
	listQuery1 := query.ByField(query.GreaterThanOperator, "revision", strconv.FormatInt(revisionKnownToProxy-1, 10))
	listQuery2 := query.ByField(query.LessThanOperator, "revision", strconv.FormatInt(lastKnownRevision+1, 10))

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
