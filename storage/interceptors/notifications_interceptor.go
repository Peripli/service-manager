package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

type NotificationsInterceptor struct {
	PlatformIdSetterFunc  func(ctx context.Context, object types.Object) string
	AdditionalDetailsFunc func(ctx context.Context, object types.Object, repository storage.Repository) (json.Marshaler, error)
}

func (ni *NotificationsInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return h
}

func (ni *NotificationsInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return h
}

func (ni *NotificationsInterceptor) AroundTxDelete(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return h
}

func (ni *NotificationsInterceptor) OnTxCreate(h storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, repository storage.Repository, newObject types.Object) error {
		if err := h(ctx, repository, newObject); err != nil {
			return err
		}

		additionalDetails, err := ni.AdditionalDetailsFunc(ctx, newObject, repository)
		if err != nil {
			return err
		}

		platformID := ni.PlatformIdSetterFunc(ctx, newObject)

		return createNotification(ctx, repository, types.CREATED, newObject.GetType(), platformID, types.Payload{
			New: &types.ObjectPayload{
				Resource:   newObject,
				Additional: additionalDetails,
			},
		})
	}
}

func (ni *NotificationsInterceptor) OnTxUpdate(h storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, repository storage.Repository, oldObject, newObject types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		updatedObject, err := h(ctx, repository, oldObject, newObject, labelChanges...)
		if err != nil {
			return nil, err
		}

		additionalDetails, err := ni.AdditionalDetailsFunc(ctx, updatedObject, repository)
		if err != nil {
			return nil, err
		}

		oldPlatformID := ni.PlatformIdSetterFunc(ctx, oldObject)
		newPlatformID := ni.PlatformIdSetterFunc(ctx, newObject)

		if oldPlatformID != newPlatformID {
			if err := createNotification(ctx, repository, types.CREATED, updatedObject.GetType(), newPlatformID, types.Payload{
				New: &types.ObjectPayload{
					Resource:   updatedObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}

			if err := createNotification(ctx, repository, types.DELETED, updatedObject.GetType(), oldPlatformID, types.Payload{
				Old: &types.ObjectPayload{
					Resource:   oldObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}
		}

		if err := createNotification(ctx, repository, types.MODIFIED, updatedObject.GetType(), newPlatformID, types.Payload{
			New: &types.ObjectPayload{
				Resource:   updatedObject,
				Additional: additionalDetails,
			},
			Old: &types.ObjectPayload{
				Resource:   oldObject,
				Additional: additionalDetails,
			},
			LabelChanges: labelChanges,
		}); err != nil {
			return nil, err
		}

		return updatedObject, nil
	}
}

func (ni *NotificationsInterceptor) OnTxDelete(h storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
	return func(ctx context.Context, repository storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		additionalDetailsMap := make(map[string]json.Marshaler)

		for i := 0; i < objects.Len(); i++ {
			object := objects.ItemAt(i)
			additionalDetails, err := ni.AdditionalDetailsFunc(ctx, object, repository)
			if err != nil {
				return nil, err
			}

			additionalDetailsMap[object.GetID()] = additionalDetails
		}

		deletedObjects, err := h(ctx, repository, objects, deletionCriteria...)
		if err != nil {
			return nil, err
		}

		for i := 0; i < deletedObjects.Len(); i++ {
			oldObject := deletedObjects.ItemAt(i)

			platformID := ni.PlatformIdSetterFunc(ctx, oldObject)

			if err := createNotification(ctx, repository, types.DELETED, oldObject.GetType(), platformID, types.Payload{
				Old: &types.ObjectPayload{
					Resource:   oldObject,
					Additional: additionalDetailsMap[oldObject.GetID()],
				},
			}); err != nil {
				return nil, err
			}
		}

		return deletedObjects, nil
	}
}

func createNotification(ctx context.Context, repository storage.Repository, op types.OperationType, resource types.ObjectType, platformID string, payload types.Payload) error {
	UUID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate GUID for notification of type %s for resource of type %s: %s", op, resource, err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	payloadBytes, err = sjson.DeleteBytes(payloadBytes, "old.resource.credentials")
	if err != nil {
		return err
	}

	payloadBytes, err = sjson.DeleteBytes(payloadBytes, "new.resource.credentials")
	if err != nil {
		return err
	}

	currentTime := time.Now()

	notification := &types.Notification{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			Labels:    map[string][]string{},
		},
		Resource:   resource,
		Type:       op,
		PlatformID: platformID,
		Payload:    payloadBytes,
	}

	notificationID, err := repository.Create(ctx, notification)
	if err != nil {
		return err
	}
	log.C(ctx).Debugf("Successfully created notification with id %s of type %s for resource type %s", notificationID, notification.Type, notification.Resource)

	return nil
}
