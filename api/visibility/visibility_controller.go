package visibility

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/query"

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

	var visibilityID string
	err = c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		logger.Debugf("Creating visibility and labels...")
		visibilityID, err = storage.Visibility().Create(ctx, visibility)
		return err
	})
	if err != nil {
		return nil, util.HandleStorageError(err, "visibility")
	}

	logger.Errorf("new service visibility id is %s", visibilityID)
	return util.NewJSONResponse(http.StatusCreated, visibility)
}

func (c *Controller) getVisibility(r *web.Request) (*web.Response, error) {
	visibilityID := r.PathParams[reqVisibilityID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting visibility with id %s", visibilityID)

	visibility, err := c.Repository.Visibility().Get(ctx, visibilityID)
	if err = util.HandleStorageError(err, "visibility"); err != nil {
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

	if p.ID != "" {
		byPlatformID := query.ByField(query.EqualsOrNilOperator, "platform_id", p.ID)
		if ctx, err = query.AddCriteria(ctx, byPlatformID); err != nil {
			return nil, util.HandleSelectionError(err)
		}
		r.Request = r.WithContext(ctx)
	}
	visibilities, err = c.Repository.Visibility().List(ctx, query.CriteriaForContext(ctx)...)
	if err != nil {
		return nil, util.HandleSelectionError(err)
	}
	return util.NewJSONResponse(http.StatusOK, types.Visibilities{
		Visibilities: visibilities,
	})
}

func (c *Controller) deleteAllVisibilities(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting visibilities...")

	if err := c.Repository.Visibility().Delete(ctx, query.CriteriaForContext(ctx)...); err != nil {
		return nil, util.HandleSelectionError(err, "visibility")
	}
	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *Controller) deleteVisibility(r *web.Request) (*web.Response, error) {
	visibilityID := r.PathParams[reqVisibilityID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting visibility with id %s", visibilityID)

	byIDQuery := query.ByField(query.EqualsOperator, "id", visibilityID)
	if err := c.Repository.Visibility().Delete(ctx, byIDQuery); err != nil {
		return nil, util.HandleStorageError(err, "visibility")
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *Controller) patchVisibility(r *web.Request) (*web.Response, error) {
	visibilityID := r.PathParams[reqVisibilityID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating visibility  with id %s", visibilityID)

	visibility, err := c.Repository.Visibility().Get(ctx, visibilityID)
	if err != nil {
		return nil, util.HandleStorageError(err, "visibility")
	}

	createdAt := visibility.CreatedAt

	changes, err := query.LabelChangesFromJSON(r.Body)
	if err != nil {
		return nil, util.HandleLabelChangeError(err)
	}
	if r.Body, err = sjson.DeleteBytes(r.Body, "labels"); err != nil {
		return nil, err
	}

	if err := util.BytesToObject(r.Body, visibility); err != nil {
		return nil, err
	}

	visibility.ID = visibilityID
	visibility.CreatedAt = createdAt
	visibility.UpdatedAt = time.Now().UTC()

	err = c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		return storage.Visibility().Update(ctx, visibility, changes...)
	})

	if err != nil {
		return nil, util.HandleStorageError(err, "visibility")
	}

	return util.NewJSONResponse(http.StatusOK, *visibility)
}
