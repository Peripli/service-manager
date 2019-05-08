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

func (p *Payload) Validate(op types.OperationType) error {
	switch op {
	case types.CREATED:
		if p.New == nil || p.New.Resource == nil || p.New.Additional == nil {
			return fmt.Errorf("new resource and non empty additional details are required for CREATED notifications")
		}

		if err := p.New.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid new resource in CREATED notification: %s", err)
		}

		if err := p.New.Additional.Validate(); err != nil {
			return fmt.Errorf("invalid new resource additional details in CREATED notification: %s", err)
		}
	case types.MODIFIED:
		if p.Old == nil || p.Old.Resource == nil || p.Old.Additional == nil {
			return fmt.Errorf("new resource is required for MODIFIED notifications")
		}

		if err := p.Old.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid old resource in MODIFIED notification: %s", err)
		}

		if err := p.Old.Additional.Validate(); err != nil {
			return fmt.Errorf("invalid old resource additional details in MODIFIED notification: %s", err)
		}

		if p.New == nil || p.New.Resource == nil || p.New.Additional == nil {
			return fmt.Errorf("old resource is required for MODIFIED notifications")
		}

		if err := p.New.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid new resource in MODIFIED notification: %s", err)
		}

		if err := p.New.Additional.Validate(); err != nil {
			return fmt.Errorf("invalid new resource additional details in MODIFIED notification: %s", err)
		}
	case types.DELETED:
		if p.Old == nil || p.Old.Resource == nil || p.Old.Additional == nil {
			return fmt.Errorf("old resource is required for DELETED notifications")
		}

		if err := p.Old.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid new resource in DELETED notification: %s", err)
		}

		if err := p.Old.Additional.Validate(); err != nil {
			return fmt.Errorf("invalid old resource additional details in CREATED notification: %s", err)
		}
	}

	return nil
}

type ObjectPayload struct {
	Resource   types.Object        `json:"resource,omitempty"`
	Additional util.InputValidator `json:"additional,omitempty"`
}

type NotificationsInterceptor struct {
	PlatformIdSetterFunc  func(ctx context.Context, object types.Object) string
	AdditionalDetailsFunc func(ctx context.Context, object types.Object, repository storage.Repository) (util.InputValidator, error)
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

		return createNotification(ctx, repository, types.CREATED, newObject.GetType(), platformID, &Payload{
			New: &ObjectPayload{
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
			if err := createNotification(ctx, repository, types.CREATED, updatedObject.GetType(), newPlatformID, &Payload{
				New: &ObjectPayload{
					Resource:   updatedObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}

			if err := createNotification(ctx, repository, types.DELETED, updatedObject.GetType(), oldPlatformID, &Payload{
				Old: &ObjectPayload{
					Resource:   oldObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}
		}

		if err := createNotification(ctx, repository, types.MODIFIED, updatedObject.GetType(), newPlatformID, &Payload{
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

			platformID := ni.PlatformIdSetterFunc(ctx, oldObject)

			if err := createNotification(ctx, repository, types.DELETED, oldObject.GetType(), platformID, &Payload{
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

func createNotification(ctx context.Context, repository storage.Repository, op types.OperationType, resource types.ObjectType, platformID string, payload *Payload) error {
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
