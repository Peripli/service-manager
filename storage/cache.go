package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

func newSelectionCache() *Cache {
	return &Cache{
		objectLists: make(map[Key]types.ObjectList),
		objects:     make(map[Key]types.Object),
	}
}

type Key struct {
	objectType types.ObjectType
	criteria   string
}

type Cache struct {
	objectLists map[Key]types.ObjectList
	objects     map[Key]types.Object
	mu          sync.RWMutex
}

func (c *Cache) PutList(ctx context.Context, obj types.ObjectList, criteria ...query.Criterion) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if obj.Len() == 0 {
		return
	}
	key := Key{
		objectType: obj.ItemAt(0).GetType(),
		criteria:   stringifyCriteria(criteria...),
	}

	c.objectLists[key] = obj
}

func (c *Cache) Put(obj types.Object, criteria ...query.Criterion) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := Key{
		objectType: obj.GetType(),
		criteria:   stringifyCriteria(criteria...),
	}
	c.objects[key] = obj
}

func (c *Cache) InvalidateForType(objectType types.ObjectType, criteria ...query.Criterion) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.objectLists {
		if key.objectType == objectType && (len(criteria) == 0 || key.criteria == stringifyCriteria(criteria...)) {
			delete(c.objectLists, key)
		}
	}

	for key := range c.objects {
		if key.objectType == objectType && (len(criteria) == 0 || key.criteria == stringifyCriteria(criteria...)) {
			delete(c.objectLists, key)
		}
	}
}

func (c *Cache) Get(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := Key{
		objectType: objectType,
		criteria:   stringifyCriteria(criteria...),
	}
	object, found := c.objects[key]
	if found {
		log.C(ctx).Debugf("Using cached entry for object of type %s and criteria %s", objectType, key.criteria)
		return object, true
	}

	return nil, false
}

func (c *Cache) GetList(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) types.ObjectList {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := Key{
		objectType: objectType,
		criteria:   stringifyCriteria(criteria...),
	}

	list, found := c.objectLists[key]
	if found {
		log.C(ctx).Debugf("Using cached entry for object list of type %s and criteria %s", objectType, key.criteria)
	}

	return list
}

func stringifyCriteria(criteria ...query.Criterion) string {
	key := ""
	for _, criterion := range criteria {
		key += fmt.Sprintf("%s:%s:%s:%v;", criterion.Type, criterion.LeftOp, criterion.Operator, criterion.RightOp)
	}

	return key
}
