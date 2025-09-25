package events

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/lib/pq"
	"time"
)

const EVENTS_CHANNEL = "events"
const dbPingInterval = time.Second * 60

type PostgresEventListener struct {
	ctx        context.Context
	listener   *pq.Listener
	storageURI string
	callBacks  map[string]func(message *Message) error
}

func NewPostgresEventListener(ctx context.Context,
	storageURI string,
	callBacks map[string]func(message *Message) error) *PostgresEventListener {
	ps := &PostgresEventListener{ctx: ctx, storageURI: storageURI, callBacks: callBacks}

	return ps
}

type Message struct {
	Table  string
	Action string
	Data   json.RawMessage
}

func (pe *PostgresEventListener) ConnectDB() error {
	eventCallback := func(et pq.ListenerEventType, err error) {
		switch et {
		case pq.ListenerEventConnected, pq.ListenerEventReconnected:
			log.C(pe.ctx).Info("DB connection for events established")
		case pq.ListenerEventDisconnected, pq.ListenerEventConnectionAttemptFailed:
			log.C(pe.ctx).WithError(err).Error("DB connection for events closed")
		}
		if err != nil {
			log.C(pe.ctx).WithError(err).Error("Event notification error")
		}
	}
	pe.listener = pq.NewListener(pe.storageURI, 30*time.Second, time.Minute, eventCallback)
	err := pe.listener.Listen(EVENTS_CHANNEL)
	if err != nil {
		return err
	}

	go pe.waitForNotification()
	return nil
}
func (pe *PostgresEventListener) StopConnection() {
	if err := pe.listener.Close(); err != nil {
		log.C(pe.ctx).WithError(err).Error("Could not close event listener db connection")
	}
}

func (pe *PostgresEventListener) processPayload(message string) error {
	payload := &Message{}
	if err := json.Unmarshal([]byte(message), payload); err != nil {
		log.C(pe.ctx).WithError(err).Error("Could not unmarshal event notification payload.")
		return err
	}
	callBack, ok := pe.callBacks[payload.Table+"-"+payload.Action]
	if ok {
		callBack(payload)

	}
	return nil
}
func (pe *PostgresEventListener) waitForNotification() {
	for {
		select {
		case n, ok := <-pe.listener.Notify:
			{
				if !ok {
					log.C(pe.ctx).Error("Notification channel closed")
					return
				}
				if n == nil {
					log.C(pe.ctx).Debug("Empty notification received")
					continue
				}
				// to do handle error
				pe.processPayload(n.Extra)
			}
		case <-time.After(dbPingInterval):
			log.C(pe.ctx).Debugf(" Pinging connection")
			if err := pe.listener.Ping(); err != nil {
				log.C(pe.ctx).WithError(err).Error("Pinging connection failed")
				return
			}
		}
	}
}
