package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/util/slice"
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

type objectDetails map[string]util.InputValidator

type NotificationsInterceptor struct {
	PlatformIDsProviderFunc func(ctx context.Context, object types.Object, repository storage.Repository) ([]string, error)
	AdditionalDetailsFunc   func(ctx context.Context, objects types.ObjectList, repository storage.Repository) (objectDetails, error)
}

func (ni *NotificationsInterceptor) OnTxCreate(h storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, repository storage.Repository, obj types.Object) (types.Object, error) {
		newObj, err := h(ctx, repository, obj)
		if err != nil {
			return nil, err
		}

		additionalDetails, err := ni.AdditionalDetailsFunc(ctx, types.NewObjectArray(obj), repository)
		if err != nil {
			return nil, err
		}

		platformIDs, err := ni.PlatformIDsProviderFunc(ctx, obj, repository)
		if err != nil {
			return nil, err
		}

		for _, platformID := range platformIDs {
			if err := CreateNotification(ctx, repository, types.CREATED, newObj.GetType(), platformID, &Payload{
				New: &ObjectPayload{
					Resource:   newObj,
					Additional: additionalDetails[obj.GetID()],
				},
			}); err != nil {
				return nil, err
			}
		}

		return newObj, nil
	}
}

func (ni *NotificationsInterceptor) OnTxUpdate(h storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, repository storage.Repository, oldObject, newObject types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		oldPlatformIDs, err := ni.PlatformIDsProviderFunc(ctx, oldObject, repository)
		if err != nil {
			return nil, err
		}

		updatedObject, err := h(ctx, repository, oldObject, newObject, labelChanges...)
		if err != nil {
			return nil, err
		}

		detailsMap, err := ni.AdditionalDetailsFunc(ctx, types.NewObjectArray(updatedObject), repository)
		if err != nil {
			return nil, err
		}
		additionalDetails := detailsMap[updatedObject.GetID()]

		updatedPlatformIDs, err := ni.PlatformIDsProviderFunc(ctx, updatedObject, repository)
		if err != nil {
			return nil, err
		}

		preexistingPlatformIDs, addedPlatformIDs, removedPlatformIDs := determinePlatformIDs(oldPlatformIDs, updatedPlatformIDs)

		oldObjectLabels := oldObject.GetLabels()
		updatedObjectLabels := updatedObject.GetLabels()

		if updatedObject.Equals(oldObject) {
			updatedObject.SetLabels(nil)
		}
		oldObject.SetLabels(nil)

		for _, platformID := range addedPlatformIDs {
			if err := CreateNotification(ctx, repository, types.CREATED, updatedObject.GetType(), platformID, &Payload{
				New: &ObjectPayload{
					Resource:   updatedObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}
		}

		for _, platformID := range removedPlatformIDs {
			if err := CreateNotification(ctx, repository, types.DELETED, updatedObject.GetType(), platformID, &Payload{
				Old: &ObjectPayload{
					Resource:   oldObject,
					Additional: additionalDetails,
				},
			}); err != nil {
				return nil, err
			}
		}

		modifiedPlatformIDs := append(preexistingPlatformIDs, addedPlatformIDs...)
		for _, platformID := range modifiedPlatformIDs {
			if err := CreateNotification(ctx, repository, types.MODIFIED, updatedObject.GetType(), platformID, &Payload{
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
		}

		oldObject.SetLabels(oldObjectLabels)
		updatedObject.SetLabels(updatedObjectLabels)

		return updatedObject, nil
	}
}

func (ni *NotificationsInterceptor) OnTxDelete(h storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
	return func(ctx context.Context, repository storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
		objectIDPlatformsMap := make(map[string][]string)
		for i := 0; i < objects.Len(); i++ {
			oldObject := objects.ItemAt(i)

			platformIDs, err := ni.PlatformIDsProviderFunc(ctx, oldObject, repository)
			if err != nil {
				return err
			}

			objectIDPlatformsMap[oldObject.GetID()] = platformIDs
		}

		additionalDetails, err := ni.AdditionalDetailsFunc(ctx, objects, repository)
		if err != nil {
			return err
		}

		if err := h(ctx, repository, objects, deletionCriteria...); err != nil {
			return err
		}

		for i := 0; i < objects.Len(); i++ {
			oldObject := objects.ItemAt(i)

			platformIDs := objectIDPlatformsMap[oldObject.GetID()]

			for _, platformID := range platformIDs {
				if err := CreateNotification(ctx, repository, types.DELETED, oldObject.GetType(), platformID, &Payload{
					Old: &ObjectPayload{
						Resource:   oldObject,
						Additional: additionalDetails[oldObject.GetID()],
					},
				}); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

func CreateNotification(ctx context.Context, repository storage.Repository, op types.NotificationOperation, resource types.ObjectType, platformID string, payload *Payload) error {
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
		Resource:      resource,
		Type:          op,
		PlatformID:    platformID,
		Payload:       payloadBytes,
		CorrelationID: log.CorrelationIDFromContext(ctx),
	}

	createdNotification, err := repository.Create(ctx, notification)
	if err != nil {
		return err
	}
	log.C(ctx).Debugf("Successfully created notification with id %s of type %s for resource type %s", createdNotification.GetID(), notification.Type, notification.Resource)

	return nil
}

func determinePlatformIDs(oldPlatformIDs, updatedPlatformIDs []string) ([]string, []string, []string) {
	preexistingPlatformIDs := slice.StringsIntersection(oldPlatformIDs, updatedPlatformIDs)
	addedPlatformIDs := findDistinctStrings(updatedPlatformIDs, preexistingPlatformIDs)
	removedPlatformIDs := findDistinctStrings(oldPlatformIDs, preexistingPlatformIDs)

	return preexistingPlatformIDs, addedPlatformIDs, removedPlatformIDs
}

func findDistinctStrings(str1, str2 []string) []string {
	distinct := make([]string, 0)
	for _, s1 := range str1 {
		isDistinct := true
		for _, s2 := range str2 {
			if s1 == s2 {
				isDistinct = false
				break
			}
		}

		if isDistinct {
			distinct = append(distinct, s1)
		}
	}

	return distinct
}
