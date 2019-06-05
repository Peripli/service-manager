package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

type Payload struct {
	New          *ObjectPayload     `json:"new,omitempty"`
	Old          *ObjectPayload     `json:"old,omitempty"`
	LabelChanges query.LabelChanges `json:"label_changes,omitempty"`
}

type ObjectPayload struct {
	Resource   types.Object        `json:"resource,omitempty"`
	Additional util.InputValidator `json:"additional,omitempty"`
}

type NotificationsInterceptor struct {
	PlatformIdProviderFunc func(ctx context.Context, object types.Object) string
	AdditionalDetailsFunc  func(ctx context.Context, object types.Object, repository storage.Repository) (util.InputValidator, error)
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
	return func(ctx context.Context, repository storage.Repository, obj types.Object) (types.Object, error) {
		newObj, err := h(ctx, repository, obj)
		if err != nil {
			return nil, err
		}

		additionalDetails, err := ni.AdditionalDetailsFunc(ctx, obj, repository)
		if err != nil {
			return nil, err
		}

		platformID := ni.PlatformIdProviderFunc(ctx, newObj)

		return newObj, CreateNotification(ctx, repository, types.CREATED, newObj.GetType(), platformID, &Payload{
			New: &ObjectPayload{
				Resource:   newObj,
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

		oldPlatformID := ni.PlatformIdProviderFunc(ctx, oldObject)
		newPlatformID := ni.PlatformIdProviderFunc(ctx, newObject)

		// if the resource update contains change in the platform ID field this means that the notification would be processed by
		// two platforms - one needs to perform a delete operation and the other needs to perform a create operation.
		if oldPlatformID != newPlatformID {
			if err := CreateNotification(ctx, repository, types.CREATED, updatedObject.GetType(), newPlatformID, &Payload{
				New: &ObjectPayload{
					Resource:   updatedObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}

			if err := CreateNotification(ctx, repository, types.DELETED, updatedObject.GetType(), oldPlatformID, &Payload{
				Old: &ObjectPayload{
					Resource:   oldObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}
		}

		if err := CreateNotification(ctx, repository, types.MODIFIED, updatedObject.GetType(), newPlatformID, &Payload{
			New: &ObjectPayload{
				Resource:   updatedObject,
				Additional: additionalDetails,
			},
			Old: &ObjectPayload{
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
		additionalDetailsMap := make(map[string]util.InputValidator)

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

			platformID := ni.PlatformIdProviderFunc(ctx, oldObject)

			if err := CreateNotification(ctx, repository, types.DELETED, oldObject.GetType(), platformID, &Payload{
				Old: &ObjectPayload{
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

func CreateNotification(ctx context.Context, repository storage.Repository, op types.OperationType, resource types.ObjectType, platformID string, payload *Payload) error {
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
