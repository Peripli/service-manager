package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-manager/pkg/util/slice"

	"github.com/Peripli/service-manager/storage/interceptors"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/agent/platform"
)

type visibilityPayload struct {
	New          visibilityWithAdditionalDetails `json:"new"`
	Old          visibilityWithAdditionalDetails `json:"old"`
	LabelChanges query.LabelChanges              `json:"label_changes"`
}

type visibilityWithAdditionalDetails struct {
	Resource   *types.Visibility                 `json:"resource"`
	Additional interceptors.VisibilityAdditional `json:"additional"`
}

// Validate validates the visibility payload
func (vp visibilityPayload) Validate(op types.OperationType) error {
	switch op {
	case types.CREATED:
		if err := vp.New.Validate(); err != nil {
			return err
		}
	case types.MODIFIED:
		if err := vp.Old.Validate(); err != nil {
			return err
		}
		if err := vp.New.Validate(); err != nil {
			return err
		}
	case types.DELETED:
		if err := vp.Old.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the visibility details
func (vwad visibilityWithAdditionalDetails) Validate() error {
	if vwad.Resource == nil {
		return fmt.Errorf("resource in notification payload cannot be nil")
	}

	if vwad.Resource.ID == "" {
		return fmt.Errorf("visibility id cannot be empty")
	}

	if vwad.Resource.ServicePlanID == "" {
		return fmt.Errorf("visibility service plan id cannot be empty")
	}

	return vwad.Additional.Validate()
}

// VisibilityResourceNotificationsHandler handles notifications for visibilities
type VisibilityResourceNotificationsHandler struct {
	VisibilityClient platform.VisibilityClient

	ProxyPrefix     string
	BrokerBlacklist []string
}

// OnCreate creates visibilities from the specified notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnCreate(ctx context.Context, payload json.RawMessage) {
	if vnh.VisibilityClient == nil {
		log.C(ctx).Warn("Platform client cannot handle visibilities. Visibility notification will be skipped")
		return
	}

	log.C(ctx).Debugf("Processing visibility create notification with payload %s...", string(payload))

	visPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling visibility create notification payload")
		return
	}

	if err := visPayload.Validate(types.CREATED); err != nil {
		log.C(ctx).WithError(err).Error("error validating visibility payload")
		return
	}

	v := visPayload.New

	if slice.StringsAnyEquals(vnh.BrokerBlacklist, v.Additional.BrokerName) {
		log.C(ctx).Infof("Broker name %s for the visibility create notification is part of broker blacklist. Skipping notification...", v.Additional.BrokerName)
		return
	}

	platformBrokerName := vnh.brokerProxyName(v.Additional.BrokerName, v.Additional.BrokerID)

	log.C(ctx).Infof("Attempting to enable access for plan with catalog ID %s for platform broker with name %s and labels %v...", v.Additional.ServicePlan.CatalogID, platformBrokerName, v.Resource.GetLabels())

	if err := vnh.VisibilityClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: v.Additional.ServicePlan.CatalogID,
		Labels:        v.Resource.GetLabels(),
	}); err != nil {
		log.C(ctx).WithError(err).Errorf("error enabling access for plan %s in broker with name %s", v.Additional.ServicePlan.CatalogID, platformBrokerName)
		return
	}
	log.C(ctx).Infof("Successfully enabled access for plan with catalog ID %s for platform broker with name %s and labels %v...", v.Additional.ServicePlan.CatalogID, platformBrokerName, v.Resource.GetLabels())
}

// OnUpdate modifies visibilities from the specified notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnUpdate(ctx context.Context, payload json.RawMessage) {
	if vnh.VisibilityClient == nil {
		log.C(ctx).Warn("Platform client cannot handle visibilities. Visibility notification will be skipped.")
		return
	}

	log.C(ctx).Debugf("Processing visibility update notification with payload %s...", string(payload))

	visibilityPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visibilityPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling visibility create notification payload")
		return
	}

	if err := visibilityPayload.Validate(types.MODIFIED); err != nil {
		log.C(ctx).WithError(err).Error("error validating visibility payload")
		return
	}

	oldVisibilityPayload := visibilityPayload.Old
	newVisibilityPayload := visibilityPayload.New

	if slice.StringsAnyEquals(vnh.BrokerBlacklist, oldVisibilityPayload.Additional.BrokerName) {
		log.C(ctx).Infof("Broker name %s for the visibility update notification is part of broker blacklist. Skipping notification...", oldVisibilityPayload.Additional.BrokerName)
		return
	}

	platformBrokerName := vnh.brokerProxyName(oldVisibilityPayload.Additional.BrokerName, oldVisibilityPayload.Additional.BrokerID)

	labelsToAdd, labelsToRemove := LabelChangesToLabels(visibilityPayload.LabelChanges)

	if oldVisibilityPayload.Additional.ServicePlan.CatalogID != newVisibilityPayload.Additional.ServicePlan.CatalogID {
		log.C(ctx).Infof("The catalog plan ID has been modified. Attempting to disable access for plan with catalog ID %s for platform broker with name %s and labels %v...", oldVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, oldVisibilityPayload.Resource.GetLabels())

		if err := vnh.VisibilityClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
			BrokerName:    platformBrokerName,
			CatalogPlanID: oldVisibilityPayload.Additional.ServicePlan.CatalogID,
			Labels:        oldVisibilityPayload.Resource.GetLabels(),
		}); err != nil {
			log.C(ctx).WithError(err).Errorf("error disabling access for plan %s in broker with name %s", oldVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName)
			return
		}
		log.C(ctx).Infof("Successfully disabled access for plan with catalog ID %s for platform broker with name %s and labels %v...", oldVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, oldVisibilityPayload.Resource.GetLabels())

		log.C(ctx).Infof("The catalog plan ID has been modified. Attempting to enable access for plan with catalog ID %s for platform broker with name %s and labels %v...", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, newVisibilityPayload.Resource.GetLabels())

		if err := vnh.VisibilityClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
			BrokerName:    platformBrokerName,
			CatalogPlanID: newVisibilityPayload.Additional.ServicePlan.CatalogID,
			Labels:        newVisibilityPayload.Resource.GetLabels(),
		}); err != nil {
			log.C(ctx).WithError(err).Errorf("error enabling access for plan %s in broker with name %s", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName)
			return
		}
		log.C(ctx).Infof("Successfully enabled access for plan with catalog ID %s for platform broker with name %s and labels %v...", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, newVisibilityPayload.Resource.GetLabels())
	}

	if err := vnh.enableServiceAccess(ctx, labelsToAdd, newVisibilityPayload, platformBrokerName); err != nil {
		return
	}

	if err := vnh.disableServiceAccess(ctx, labelsToRemove, newVisibilityPayload, platformBrokerName); err != nil {
		return
	}
}

func (vnh *VisibilityResourceNotificationsHandler) disableServiceAccess(ctx context.Context, labelsToRemove types.Labels, newVisibilityPayload visibilityWithAdditionalDetails, platformBrokerName string) error {
	if (len(labelsToRemove) == 0 && newVisibilityPayload.Resource.PlatformID == "") || (len(labelsToRemove) != 0 && newVisibilityPayload.Resource.PlatformID != "") {
		log.C(ctx).Infof("Attempting to disable access for plan with catalog ID %s for platform broker with name %s and labels %v...", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, labelsToRemove)

		if err := vnh.VisibilityClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
			BrokerName:    platformBrokerName,
			CatalogPlanID: newVisibilityPayload.Additional.ServicePlan.CatalogID,
			Labels:        labelsToRemove,
		}); err != nil {
			log.C(ctx).WithError(err).Errorf("error disabling access for plan %s in broker with name %s", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName)
			return err
		}
		log.C(ctx).Infof("Successfully disabled access for plan with catalog ID %s for platform broker with name %s and labels %v...", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, labelsToRemove)
	}
	return nil
}

func (vnh *VisibilityResourceNotificationsHandler) enableServiceAccess(ctx context.Context, labelsToAdd types.Labels, newVisibilityPayload visibilityWithAdditionalDetails, platformBrokerName string) error {
	if (len(labelsToAdd) == 0 && newVisibilityPayload.Resource.PlatformID == "") || (len(labelsToAdd) != 0 && newVisibilityPayload.Resource.PlatformID != "") {
		log.C(ctx).Infof("Attempting to enable access for plan with catalog ID %s for platform broker with name %s and labels %v...", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, labelsToAdd)

		if err := vnh.VisibilityClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
			BrokerName:    platformBrokerName,
			CatalogPlanID: newVisibilityPayload.Additional.ServicePlan.CatalogID,
			Labels:        labelsToAdd,
		}); err != nil {
			log.C(ctx).WithError(err).Errorf("error enabling access for plan %s in broker with name %s", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName)
			return err
		}
		log.C(ctx).Infof("Successfully enabled access for plan with catalog ID %s for platform broker with name %s and labels %v...", newVisibilityPayload.Additional.ServicePlan.CatalogID, platformBrokerName, labelsToAdd)
	}
	return nil
}

// OnDelete deletes visibilities from the provided notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnDelete(ctx context.Context, payload json.RawMessage) {
	if vnh.VisibilityClient == nil {
		log.C(ctx).Warn("Platform client cannot handle visibilities. Visibility notification will be skipped")
		return
	}

	log.C(ctx).Debugf("Processing visibility delete notification with payload %s...", string(payload))

	visibilityPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visibilityPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling visibility delete notification payload")
		return
	}

	if err := visibilityPayload.Validate(types.DELETED); err != nil {
		log.C(ctx).WithError(err).Error("error validating visibility payload")
		return
	}

	v := visibilityPayload.Old

	if slice.StringsAnyEquals(vnh.BrokerBlacklist, v.Additional.BrokerName) {
		log.C(ctx).Infof("Broker name %s for the visibility create notification is part of broker blacklist. Skipping notification...", v.Additional.BrokerName)
		return
	}

	platformBrokerName := vnh.brokerProxyName(v.Additional.BrokerName, v.Additional.BrokerID)

	log.C(ctx).Infof("Attempting to disable access for plan with catalog ID %s for platform broker with name %s and labels %v...", v.Additional.ServicePlan.CatalogID, platformBrokerName, v.Resource.GetLabels())

	if err := vnh.VisibilityClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: v.Additional.ServicePlan.CatalogID,
		Labels:        v.Resource.GetLabels(),
	}); err != nil {
		log.C(ctx).WithError(err).Errorf("error disabling access for plan %s in broker with name %s", v.Additional.ServicePlan.CatalogID, platformBrokerName)
		return
	}
	log.C(ctx).Infof("Successfully disabled access for plan with catalog ID %s for platform broker with name %s and labels %v...", v.Additional.ServicePlan.CatalogID, platformBrokerName, v.Resource.GetLabels())

}

func (vnh *VisibilityResourceNotificationsHandler) brokerProxyName(brokerName, brokerID string) string {
	return fmt.Sprintf("%s%s-%s", vnh.ProxyPrefix, brokerName, brokerID)
}
