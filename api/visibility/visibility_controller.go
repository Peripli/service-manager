package visibility

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/selection"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gofrs/uuid"
)

const (
	reqVisibilityID = "visibility_id"
)

type Controller struct {
	Repository storage.Repository
}

var _ web.Controller = &Controller{}

func (c *Controller) createVisibility(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logger := log.C(ctx)
	logger.Debug("Creating new visibility")

	visibility := &types.Visibility{}
	if err := util.BytesToObject(r.Body, visibility); err != nil {
		return nil, err
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for visibility: %s", err)
	}

	visibility.ID = UUID.String()

	currentTime := time.Now().UTC()
	visibility.CreatedAt = currentTime
	visibility.UpdatedAt = currentTime

	for _, label := range visibility.Labels {
		label.ID = UUID.String()
		label.CreatedAt = currentTime
		label.UpdatedAt = currentTime
	}

	var visibilityID string
	err = c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		logger.Debugf("Creating visibility and labels...")
		visibilityID, err = storage.Visibility().Create(ctx, visibility)
		return err
	})
	if err != nil {
		return nil, util.HandleStorageError(err, "visibility", visibility.ID)
	}

	logger.Errorf("new service visibility id is %s", visibilityID)
	return util.NewJSONResponse(http.StatusCreated, visibility)
}

func (c *Controller) getVisibility(r *web.Request) (*web.Response, error) {
	visibilityID := r.PathParams[reqVisibilityID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting visibility with id %s", visibilityID)

	visibility, err := c.Repository.Visibility().Get(ctx, visibilityID)
	if err = util.HandleStorageError(err, "visibility", visibilityID); err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, visibility)
}

func (c *Controller) listVisibilities(r *web.Request) (*web.Response, error) {
	var visibilities []*types.Visibility
	var err error
	ctx := r.Context()
	log.C(ctx).Debug("Getting all visibilities")

	user, ok := web.UserFromContext(ctx)
	if !ok {
		return nil, errors.New("user details not found in request context")
	}

	p := &types.Platform{}

	if err := user.Data.Data(p); err != nil {
		return nil, err
	}
	criteria, err := selection.BuildCriteriaFromRequest(r)
	if err != nil {
		log.C(ctx).Error(err)
		return nil, util.HandleSelectionError(err)
	}
	if p.ID != "" {
		platformIdCriterion := selection.Criterion{
			Type:     selection.FieldQuery,
			LeftOp:   "platform_id",
			RightOp:  []string{p.ID},
			Operator: selection.EqualsOrNilOperator,
		}
		if err := criteria.Add(platformIdCriterion); err != nil {
			return nil, util.HandleSelectionError(err)
		}
	}
	visibilities, err = c.Repository.Visibility().List(ctx, criteria...)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}
	return util.NewJSONResponse(http.StatusOK, types.Visibilities{
		Visibilities: visibilities,
	})
}

func (c *Controller) deleteVisibility(r *web.Request) (*web.Response, error) {
	visibilityID := r.PathParams[reqVisibilityID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting visibility with id %s", visibilityID)

	if err := c.Repository.Visibility().Delete(ctx, visibilityID); err != nil {
		return nil, util.HandleStorageError(err, "visibility", visibilityID)
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *Controller) patchVisibility(r *web.Request) (*web.Response, error) {
	visibilityID := r.PathParams[reqVisibilityID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating visibility  with id %s", visibilityID)

	visibility, err := c.Repository.Visibility().Get(ctx, visibilityID)
	if err != nil {
		return nil, util.HandleStorageError(err, "visibility", visibilityID)
	}

	createdAt := visibility.CreatedAt

	if err := util.BytesToObject(r.Body, visibility); err != nil {
		return nil, err
	}

	visibility.ID = visibilityID
	visibility.CreatedAt = createdAt
	visibility.UpdatedAt = time.Now().UTC()

	if err := c.Repository.Visibility().Update(ctx, visibility); err != nil {
		return nil, util.HandleStorageError(err, "visibility", visibilityID)
	}

	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, visibility)
}
