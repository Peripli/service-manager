package notifications

import (
	"context"
	"errors"
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const (
	LastKnownRevisionHeader     = "last_notification_revision"
	LastKnownRevisionQueryParam = "last_notification_revision"
	AgentVersionHeader          = "X-Peripli-Agent-Version"
)

func (c *Controller) handleWS(req *web.Request) (*web.Response, error) {
	ctx := req.Context()
	logger := log.C(ctx)
	user, ok := web.UserFromContext(req.Context())
	if !ok {
		return nil, errors.New("user details not found in request context")
	}

	platform, err := extractPlatformFromContext(user)
	if err != nil {
		return nil, err
	}
	version := req.Header.Get(AgentVersionHeader)
	if platform.Version != version {
		platform.Version = version
		_, err = c.repository.Update(ctx, platform, nil)
		if err != nil {
			logger.Errorf("Could not update platform version for platform %s: %v", platform.ID, err)
			return nil, err
		}
	}

	if user.Name == platform.Credentials.Basic.Username && !platform.CredentialsActive {
		logger.Debugf("Activating credentials for platform %s", platform.ID)
		platform.CredentialsActive = true
		platform.OldCredentials = nil
		_, err = c.repository.Update(ctx, platform, nil)
		if err != nil {
			logger.Errorf("Could not activate credentials for platform %s: %v", platform.ID, err)
			return nil, err
		}
	}
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

	notificationQueue, lastKnownToSMRevision, err := c.notificator.RegisterConsumer(platform, revisionKnownToProxy)
	if err != nil {
		if err == util.ErrInvalidNotificationRevision {
			return util.NewJSONResponse(http.StatusGone, nil)
		}
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
	if lastKnownToSMRevision != types.InvalidRevision {
		responseHeaders.Add(LastKnownRevisionHeader, strconv.FormatInt(lastKnownToSMRevision, 10))
	}

	conn, err := c.upgrade(childCtx, c.repository, platform, rw, req.Request, responseHeaders)
	if err != nil {
		c.unregisterConsumer(ctx, notificationQueue)
		return nil, err
	}

	done := make(chan struct{}, 2)

	go c.closeConn(childCtx, childCtxCancel, conn, done)
	go c.writeLoop(childCtx, conn, notificationQueue, done)
	go c.readLoop(childCtx, c.repository, platform, conn, done)

	return &web.Response{}, nil
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
