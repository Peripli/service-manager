package actions

import (
	"context"
	"errors"
	"github.com/Peripli/service-manager/pkg/types"
	"time"
)

type Notification struct {
	Entity types.Object
	Err    error
}

type ChanItem struct {
	Channel     chan Notification
	Duration    time.Duration
	ChanContext context.Context
}

type SyncEventBus struct {
	scheduledOperations map[string][]ChanItem
}

func (se *SyncEventBus) removeFromEventBus(id string, chanHolder ChanItem) {
	if _, ok := se.scheduledOperations[id]; ok {
		for i := range se.scheduledOperations[id] {
			if se.scheduledOperations[id][i] == chanHolder {
				se.scheduledOperations[id] = append(se.scheduledOperations[id][:i], se.scheduledOperations[id][i+1:]...)
				break
			}
		}
	}
}

func (se *SyncEventBus) AddListener(id string, objectsChan chan Notification, ctx context.Context) {

	if se.scheduledOperations == nil {
		se.scheduledOperations = make(map[string][]ChanItem)
	}

	chanItem := ChanItem{
		Channel:     objectsChan,
		Duration:    10 * time.Second,
		ChanContext: nil,
	}

	go se.withChannelWatch(id, chanItem, ctx)

	if _, ok := se.scheduledOperations[id]; ok {
		se.scheduledOperations[id] = append(se.scheduledOperations[id], chanItem)
	} else {
		se.scheduledOperations[id] = []ChanItem{chanItem}
	}

	print(se.scheduledOperations[id])
}

func (se *SyncEventBus) withChannelWatch(indexId string, chanItem ChanItem, ctx context.Context) {
	maxExecutionTime := time.NewTicker(chanItem.Duration)
	defer maxExecutionTime.Stop()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			se.NotifyCompleted(indexId, Notification{
				Entity: nil,
				Err:    errors.New("the context is done, either because SM crashed/exited or because action timeout elapsed"),
			})
			se.removeFromEventBus(indexId, chanItem)
			return
		case <-maxExecutionTime.C:
			se.NotifyCompleted(indexId, Notification{
				Entity: nil,
				Err:    errors.New("the maximum execution time for this even has been reached"),
			})
			se.removeFromEventBus(indexId, chanItem)
			return
		}
	}
}

func (se *SyncEventBus) NotifyCompleted(id string, object Notification) {
	if _, ok := se.scheduledOperations[id]; ok {
		for _, handler := range se.scheduledOperations[id] {
			go func(handler chan Notification) {
				handler <- object
			}(handler.Channel)
		}
	}
}