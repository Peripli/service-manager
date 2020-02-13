package notifications

import (
	"context"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"net/http"
	"strconv"
	"time"

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

func (c *Controller) handleWS(req *web.Request) (*web.Response, error) {
	ctx := req.Context()
	logger := log.C(ctx)

	revisionKnownToProxy := types.InvalidRevision
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

	// We check that a revision is valid only if its value is not types.InvalidRevision since in that case is never valid
	if revisionKnownToProxy != types.InvalidRevision {
		isProxyRevisionValid, err := c.notificator.IsRevisionValid(revisionKnownToProxy)
		if err != nil {
			return nil, err
		}
		if !isProxyRevisionValid {
			return util.NewJSONResponse(http.StatusGone, nil)
		}
	}

	user, ok := web.UserFromContext(req.Context())
	if !ok {
		return nil, errors.New("user details not found in request context")
	}

	platform, err := extractPlatformFromContext(user)
	if err != nil {
		return nil, err
	}

	revisionKnownToSM, err := c.notificator.GetLastRevision()
	if err != nil {
		return nil, err
	}

	correlationID := logger.Data[log.FieldCorrelationID].(string)
	childCtx, childCtxCancel := newContextWithCorrelationID(c.baseCtx, correlationID)

	defer func() {
		if err := recover(); err != nil {
			log.C(childCtx).Errorf("recovered from panic while establishing websocket connection: %s", err)
		}
	}()

	rw := req.HijackResponseWriter()
	responseHeaders := http.Header{}
	if revisionKnownToSM != types.InvalidRevision {
		responseHeaders.Add(LastKnownRevisionHeader, strconv.FormatInt(revisionKnownToSM, 10))
	}

	conn, err := c.upgrade(childCtx, c.repository, platform, rw, req.Request, responseHeaders)
	if err != nil {
		return nil, err
	}

	c.ensureHealthyWsConnection(childCtx, c.repository, platform, conn)

	// TODO: If the check annotated with 'TODO' in the func RegisterConsumer is removed then the error handling here will be obsolete
	notificationQueue, err := c.notificator.RegisterConsumer(platform, revisionKnownToProxy, revisionKnownToSM)
	if err != nil {
		if err == util.ErrInvalidNotificationRevision {
			conn.Close()
			return util.NewJSONResponse(http.StatusGone, nil)
		}
		conn.Close()
		return nil, err
	}

	done := make(chan struct{}, 2)

	go c.closeConn(childCtx, childCtxCancel, conn, done)
	go c.writeLoop(childCtx, conn, notificationQueue, done)
	go c.readLoop(childCtx, c.repository, platform, conn, done)

	return &web.Response{}, nil
}

func (c *Controller) ensureHealthyWsConnection(ctx context.Context, repository storage.TransactionalRepository, platform *types.Platform, conn *websocket.Conn) error {
	ch := make(chan error)
	conn.SetPingHandler(func(message string) error {

		if err := conn.SetReadDeadline(time.Now().Add(c.wsSettings.PingTimeout)); err != nil {
			ch <- err
			return err
		}

		if err := updatePlatformStatus(ctx, repository, platform.ID, true); err != nil {
			return err
		}

		err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(c.wsSettings.WriteTimeout))
		if err != nil {
			log.C(ctx).Errorf("initial pong failed: %s", err)
			ch <- err
			return err
		}
		return nil
	})
	go func() {
		<-ch
		<-ctx.Done()
		log.C(ctx).Info("Context cancelled. Terminating notifications handler")
	}()
	return nil
}

func (c *Controller) writeLoop(ctx context.Context, conn *websocket.Conn, q storage.NotificationQueue, done chan<- struct{}) {
	defer func() {
		if err := recover(); err != nil {
			log.C(ctx).Errorf("recovered from panic while writing to websocket connection: %s", err)
		}
	}()

	defer func() {
		done <- struct{}{}
	}()
	defer c.unregisterConsumer(ctx, q)

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

func (c *Controller) readLoop(ctx context.Context, repository storage.TransactionalRepository, platform *types.Platform, conn *websocket.Conn, done chan<- struct{}) {
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
			log.C(ctx).WithError(err).Error("ws: could not read")
			if err = updatePlatformStatus(ctx, repository, platform.ID, false); err != nil {
				log.C(ctx).WithError(err).Error("could not update platform status")
			}
			return
		}
	}
}

func (c *Controller) sendWsMessage(ctx context.Context, conn *websocket.Conn, msg interface{}) bool {
	if err := conn.SetWriteDeadline(time.Now().Add(c.wsSettings.WriteTimeout)); err != nil {
		log.C(ctx).WithError(err).Error("Could not set write deadline")
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.C(ctx).WithError(err).Error("ws: could not write")
		return false
	}
	return true
}

func (c *Controller) unregisterConsumer(ctx context.Context, q storage.NotificationQueue) {
	if unregErr := c.notificator.UnregisterConsumer(q); unregErr != nil {
		log.C(ctx).WithError(unregErr).Errorf("Could not unregister notification consumer")
	}
}

func extractPlatformFromContext(userContext *web.UserContext) (*types.Platform, error) {
	platform := &types.Platform{}
	err := userContext.Data(platform)
	if err != nil {
		return nil, fmt.Errorf("could not get platform from user context %v", err)
	}
	if platform.ID == "" {
		return nil, errors.New("platform ID not found in user context")
	}
	return platform, nil
}

func newContextWithCorrelationID(baseCtx context.Context, correlationID string) (context.Context, context.CancelFunc) {
	entry := log.C(baseCtx).WithField(log.FieldCorrelationID, correlationID)
	newCtx := log.ContextWithLogger(baseCtx, entry)
	return context.WithCancel(newCtx)
}

func updatePlatformStatus(ctx context.Context, repository storage.TransactionalRepository, platformID string, desiredStatus bool) error {
	if err := repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		idCriteria := query.Criterion{
			LeftOp:   "id",
			Operator: query.EqualsOperator,
			RightOp:  []string{platformID},
			Type:     query.FieldQuery,
		}
		obj, err := storage.Get(ctx, types.PlatformType, idCriteria)
		if err != nil {
			return err
		}

		platform := obj.(*types.Platform)

		if platform.Active != desiredStatus {
			platform.Active = desiredStatus
			if !platform.Active {
				platform.LastActive = time.Now()
			}

			if _, err := storage.Update(ctx, platform, nil); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
