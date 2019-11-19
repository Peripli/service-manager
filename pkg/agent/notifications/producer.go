/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package notifications

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/Peripli/service-manager/api/notifications"
	"github.com/pkg/errors"

	"github.com/Peripli/service-manager/pkg/agent/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gorilla/websocket"
)

var errLastNotificationGone = errors.New("last notification revision no longer present in SM")

// Producer reads notifications coming from the websocket connection and regularly
// triggers full resync
type Producer struct {
	producerSettings ProducerSettings
	smSettings       sm.Settings

	lastNotificationRevision int64
	conn                     *websocket.Conn
	pingPeriod               time.Duration
	readTimeout              time.Duration
	url                      *url.URL
}

// ProducerSettings are the settings for the producer
type ProducerSettings struct {
	// MinPingPeriod is the minimum allowed ping period
	MinPingPeriod time.Duration `mapstructure:"min_ping_period"`
	// Reconnect delay is the time between reconnects
	ReconnectDelay time.Duration `mapstructure:"reconnect_delay"`
	// PongTimeout is the maximum time to wait between a ping and a pong
	PongTimeout time.Duration `mapstructure:"pong_timeout"`
	// PingPeriodPercentage is the percentage of actual ping period compared to the max_ping_period returned by SM
	PingPeriodPercentage int64 `mapstructure:"ping_period_percentage"`
	// MessagesQueueSize is the size of the messages queue
	MessagesQueueSize int `mapstructure:"messages_queue_size"`
	// ResyncPeriod is the time between two resyncs
	ResyncPeriod time.Duration `mapstructure:"resync_period"`
}

// Validate validates the producer settings
func (p ProducerSettings) Validate() error {
	if p.MinPingPeriod <= 0 {
		return fmt.Errorf("ProducerSettings: min ping period must be positive duration")
	}
	if p.ReconnectDelay < 0 {
		return fmt.Errorf("ProducerSettings: reconnect delay must be non-negative duration")
	}
	if p.PongTimeout <= 0 {
		return fmt.Errorf("ProducerSettings: pong time must be positive duration")
	}
	if p.PingPeriodPercentage <= 0 || p.PingPeriodPercentage >= 100 {
		return fmt.Errorf("ProducerSettings: ping period percentage must be between 0 and 100")
	}
	if p.ResyncPeriod <= 0 {
		return fmt.Errorf("ProducerSettings: Resync Period must be positive duration")
	}
	return nil
}

// DefaultProducerSettings are the default settings for the producer
func DefaultProducerSettings() *ProducerSettings {
	return &ProducerSettings{
		MinPingPeriod:        time.Second,
		ReconnectDelay:       3 * time.Second,
		PongTimeout:          2 * time.Second,
		PingPeriodPercentage: 60,
		MessagesQueueSize:    1024,
		ResyncPeriod:         12 * time.Hour,
	}
}

// Message is the payload sent by the producer
type Message struct {
	// Notification is the notification that needs to be applied on the platform
	Notification *types.Notification

	// Resync indicates that full resync is needed
	// In this case Notification is nil
	// Resync messages are written in the channel:
	// - The first message in the channel, so we always start with a resync
	// - When notifications are lost (indicated by status 410 returned by SM)
	// - On regular interval based on configuration
	Resync bool
}

// NewProducer returns a configured producer for the given settings
func NewProducer(producerSettings *ProducerSettings, smSettings *sm.Settings) (*Producer, error) {
	notificationsURL, err := buildNotificationsURL(smSettings.URL, smSettings.NotificationsAPIPath)
	if err != nil {
		return nil, err
	}
	return &Producer{
		url:                      notificationsURL,
		producerSettings:         *producerSettings,
		smSettings:               *smSettings,
		lastNotificationRevision: types.InvalidRevision,
	}, nil
}

func buildNotificationsURL(baseURL, notificationsPath string) (*url.URL, error) {
	smURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	switch smURL.Scheme {
	case "http":
		smURL.Scheme = "ws"
	case "https":
		smURL.Scheme = "wss"
	}
	smURL.Path = path.Join(smURL.Path, notificationsPath)
	return smURL, nil
}

// Start starts the producer in a new go-routine
func (p *Producer) Start(ctx context.Context, group *sync.WaitGroup) <-chan *Message {
	group.Add(1)
	messages := make(chan *Message, p.producerSettings.MessagesQueueSize)
	go p.run(ctx, messages, group)
	return messages
}

func (p *Producer) run(ctx context.Context, messages chan *Message, group *sync.WaitGroup) {
	defer group.Done()
	resyncChan := make(chan struct{})
	p.lastNotificationRevision = types.InvalidRevision
	go p.scheduleResync(ctx, resyncChan, messages)

	for {
		// Store the state of p.lastNotificationRevision as it will be changed in p.connect after reading headers
		// We need the value that was set before establishing the connection to determine whether to resync
		needResync := p.lastNotificationRevision == types.InvalidRevision
		if err := p.connect(ctx); err != nil {
			log.C(ctx).WithError(err).Error("could not connect websocket")
			if err == errLastNotificationGone { // skip reconnect delay
				p.lastNotificationRevision = types.InvalidRevision
				continue
			}
		} else {
			log.C(ctx).Info("Successfully established websocket connection")
			if needResync {
				log.C(ctx).Info("Last notification revision is gone. Triggering resync")
				resyncChan <- struct{}{}
			}
			p.conn.SetPongHandler(func(string) error {
				log.C(ctx).Debug("Received pong")
				return p.conn.SetReadDeadline(time.Now().Add(p.readTimeout))
			})

			done := make(chan struct{}, 2)
			childContext, cancelChildrenFunc := context.WithCancel(ctx)
			go p.readNotifications(childContext, messages, done)
			go p.ping(childContext, done)

			<-done
			cancelChildrenFunc()
			<-done // Wait for both go routines to finish before closing connection

			p.closeConnection(ctx)
		}
		select {
		case <-ctx.Done():
			log.C(ctx).Info("Context cancelled. Terminating notifications handler")
			return
		case <-time.After(p.producerSettings.ReconnectDelay):
			log.C(ctx).Debug("Reconnection delay elapsed. Reattempting to establish websocket connection")
		}
	}
}

func (p *Producer) scheduleResync(ctx context.Context, resyncChan chan struct{}, messages chan *Message) {
	timer := time.NewTimer(p.producerSettings.ResyncPeriod)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.C(ctx).Info("Context cancelled while waiting. Terminating resync timer.")
			return
		case <-resyncChan:
		case <-timer.C:
			log.C(ctx).Debug("Resync period has elapsed. Triggering resync.")
		}
		select {
		case <-ctx.Done():
			log.C(ctx).Info("Context cancelled while sending message. Terminating resync timer.")
			return
		case messages <- &Message{Resync: true}:
			log.C(ctx).Debugf("Sending resync message. Channel len/cap: %d/%d", len(messages), cap(messages))
		}
		timer.Stop()
		timer = time.NewTimer(p.producerSettings.ResyncPeriod)
	}
}

func (p *Producer) readNotifications(ctx context.Context, messages chan *Message, done chan<- struct{}) {
	defer func() {
		log.C(ctx).Debug("Exiting notification reader")
		done <- struct{}{}
	}()

	log.C(ctx).Debug("Starting notification reader")
	for {
		if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			log.C(ctx).WithError(err).Error("Error setting read timeout on websocket")
			return
		}
		_, bytes, err := p.conn.ReadMessage()
		if err != nil {
			log.C(ctx).WithError(err).Error("Error reading from websocket")
			return
		}
		var notification types.Notification
		if err = json.Unmarshal(bytes, &notification); err != nil {
			log.C(ctx).WithError(err).Error("Could not unmarshal WS message into a notification")
			return
		}
		log.C(ctx).Debugf("Received notification with revision %d", notification.Revision)
		select {
		case <-ctx.Done():
			log.C(ctx).Info("Context cancelled while sending message. Terminating notification reader.")
			return
		case messages <- &Message{Notification: &notification}:
			log.C(ctx).Debugf("Notification written in channel. Channel len/cap: %d/%d", len(messages), cap(messages))
		}

		p.lastNotificationRevision = notification.Revision
	}
}

func (p *Producer) ping(ctx context.Context, done chan<- struct{}) {
	defer func() {
		log.C(ctx).Debug("Exiting pinger")
		done <- struct{}{}
	}()
	log.C(ctx).Debug("Starting pinger")
	ticker := time.NewTicker(p.pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.C(ctx).Debug("Sending ping")
			if err := p.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.C(ctx).WithError(err).Error("Could not write message on the websocket")
				return
			}
		}
	}
}

func (p *Producer) connect(ctx context.Context) error {
	headers := http.Header{}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(p.smSettings.User+":"+p.smSettings.Password))
	headers.Add("Authorization", auth)

	connectURL := *p.url
	if p.lastNotificationRevision != types.InvalidRevision {
		q := connectURL.Query()
		q.Set(notifications.LastKnownRevisionQueryParam, strconv.FormatInt(p.lastNotificationRevision, 10))
		connectURL.RawQuery = q.Encode()
	}
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: p.smSettings.RequestTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: p.smSettings.SkipSSLValidation,
		},
	}

	log.C(ctx).Debugf("Connecting to %s ...", &connectURL)
	var err error
	var resp *http.Response
	p.conn, resp, err = dialer.DialContext(ctx, connectURL.String(), headers)
	if err != nil {
		if resp != nil {
			log.C(ctx).WithError(err).Errorf("Could not connect to %s: status: %d", &connectURL, resp.StatusCode)
			if resp.StatusCode == http.StatusGone {
				return errLastNotificationGone
			}
		}

		return err
	}

	// Set the new revision as well as the ping interval and read timeout
	if err := p.readResponseHeaders(ctx, resp.Header); err != nil {
		p.closeConnection(ctx)

		return err
	}

	return nil
}

func (p *Producer) readResponseHeaders(ctx context.Context, header http.Header) error {
	if p.lastNotificationRevision == types.InvalidRevision {
		if revisionStr := header.Get(notifications.LastKnownRevisionHeader); revisionStr != "" {
			revision, err := strconv.ParseInt(revisionStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid last notification revision received (%s): %v", revisionStr, err)
			}
			if revision <= 0 {
				return fmt.Errorf("invalid last notification revision received (%d)", revision)
			}
			p.lastNotificationRevision = revision
		}
	}

	maxPingPeriod, err := time.ParseDuration(header.Get(notifications.MaxPingPeriodHeader))
	if err != nil {
		return errors.Wrap(err, "invalid max ping period received")
	}
	if maxPingPeriod < p.producerSettings.MinPingPeriod {
		return fmt.Errorf("invalid max ping period (%s) must be greater than the minimum ping period (%s)", maxPingPeriod, p.producerSettings.MinPingPeriod)
	}
	p.pingPeriod = time.Duration(int64(maxPingPeriod) * p.producerSettings.PingPeriodPercentage / 100)
	p.readTimeout = p.pingPeriod + p.producerSettings.PongTimeout // should be longer than pingPeriod
	log.C(ctx).Infof("Successfully set ping period: %s pong timeout: %s after processing headers", p.pingPeriod, p.readTimeout)
	return nil
}

func (p *Producer) closeConnection(ctx context.Context) {
	if p.conn == nil {
		return
	}
	log.C(ctx).Debug("Closing websocket connection")
	if err := p.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(p.smSettings.RequestTimeout)); err != nil {
		log.C(ctx).WithError(err).Warn("Could not send close message on websocket")
	}
	if err := p.conn.Close(); err != nil {
		log.C(ctx).WithError(err).Warn("Could not close websocket connection")
	}
}
