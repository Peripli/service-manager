package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fatih/structs"
	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Types test suite")
}

var _ = Describe("Types test", func() {
	now := time.Now()
	entries := []*testCase{
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence",
			},
			baseObjectCreateFunc: createBroker,
		},
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence",
			},
			baseObjectCreateFunc: createPlatform,
		},
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence",
			},
			baseObjectCreateFunc: createVisibility,
		},
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence",
			},
			baseObjectCreateFunc: createServiceOffering,
		},
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence",
			},
			baseObjectCreateFunc: createServicePlan,
		},
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence", "Ready", "Usable",
			},
			baseObjectCreateFunc: createServiceInstance,
		},
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence",
			},
			baseObjectCreateFunc: createNotification,
		},
		{
			expectedEqual: []string{
				"Base.Labels", "Base.UpdatedAt", "Base.PagingSequence",
			},
			baseObjectCreateFunc: createOperation,
		},
	}

	for i := range entries {
		func(entry *testCase) {
			blueprint := entry.baseObjectCreateFunc(now)
			typ := blueprint.GetType().String()
			Context(typ, func() {
				var object1, object2 Object
				BeforeEach(func() {
					object1 = entry.baseObjectCreateFunc(now)
					object2 = entry.baseObjectCreateFunc(now)

				})

				changeableProps := setProps(blueprint, "")
				for k := range changeableProps {
					func(cp propChange) {
						When(fmt.Sprintf("%s is changed", cp.path), func() {
							BeforeEach(func() {
								err := setProp(object1, cp.path, cp.value)
								Expect(err).ShouldNot(HaveOccurred())
							})
							var expectedEqual bool
							for _, e := range entry.expectedEqual {
								if e == cp.path {
									expectedEqual = true
									break
								}
							}
							itmsg := "should be equal"
							if !expectedEqual {
								itmsg = "should NOT be equal"
							}

							It(itmsg, func() {
								Expect(object1.Equals(object2)).To(Equal(expectedEqual))
							})
						})
					}(changeableProps[k])
				}
			})
		}(entries[i])
	}
})

type testCase struct {
	baseObjectCreateFunc func(time.Time) Object
	expectedEqual        []string
}

type propChange struct {
	path  string
	value interface{}
}

func setProp(object Object, key string, value interface{}) error {
	paths := strings.Split(key, ".")
	s := structs.New(object)
	finalProp := s.Field(paths[0])
	paths = paths[1:]
	var found bool
	for _, p := range paths {
		finalProp, found = finalProp.FieldOk(p)
		if !found {
			return errors.New("errored")
		}
	}
	return finalProp.Set(value)
}

func setProps(object interface{}, propPath string) []propChange {
	result := make([]propChange, 0)
	s := structs.New(object)
	fields := s.Fields()
	for _, f := range fields {
		currentPath := f.Name()
		if len(propPath) != 0 {
			currentPath = fmt.Sprintf("%s.%s", propPath, f.Name())
		}
		if f.IsZero() {
			continue
		}
		if f.Kind() != reflect.Struct && f.Kind() != reflect.Ptr {
			assigned := true
			switch f.Value().(type) {
			case string:
				result = append(result, propChange{
					path:  currentPath,
					value: "changed",
				})
			case ObjectType:
				result = append(result, propChange{
					path:  currentPath,
					value: ObjectType("changed"),
				})
			case NotificationOperation:
				result = append(result, propChange{
					path:  currentPath,
					value: NotificationOperation("changed"),
				})
			case OperationCategory:
				result = append(result, propChange{
					path:  currentPath,
					value: OperationCategory("changed"),
				})
			case OperationState:
				result = append(result, propChange{
					path:  currentPath,
					value: OperationState("changed"),
				})
			case Labels:
				result = append(result, propChange{
					path: currentPath,
					value: Labels{
						"new": []string{"value"},
					},
				})
			case json.RawMessage:
				result = append(result, propChange{
					path:  currentPath,
					value: []byte("changed"),
				})
			default:
				assigned = false
			}
			if assigned {
				continue
			}
			assigned = true

			switch f.Kind() {
			case reflect.Bool:
				result = append(result, propChange{
					path:  currentPath,
					value: false,
				})
			case reflect.Int:
				result = append(result, propChange{
					path:  currentPath,
					value: int(2),
				})
			case reflect.Int32:
				result = append(result, propChange{
					path:  currentPath,
					value: int32(2),
				})
			case reflect.Int64:
				result = append(result, propChange{
					path:  currentPath,
					value: int64(2),
				})
			default:
				assigned = false
			}
			if !assigned {
				panic("Not able to assign property")
			}
		} else {
			switch f.Value().(type) {
			case time.Time:
				result = append(result, propChange{
					path:  currentPath,
					value: time.Now(),
				})
			case *bool:
				falseVal := false
				result = append(result, propChange{
					path:  currentPath,
					value: &falseVal,
				})
			default:
				val := f.Value()
				fmt.Println(val)
				result = append(result, setProps(f.Value(), currentPath)...)
			}
		}
	}
	return result
}

func createOperation(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	return &Operation{
		Base: Base{
			ID:        "id",
			Labels:    labels,
			CreatedAt: now,
			UpdatedAt: now.Add(time.Second * 10),
		},
		Description:   "description",
		Type:          OperationCategory("category"),
		State:         OperationState("state"),
		ResourceID:    "1",
		ResourceType:  "type",
		Errors:        []byte("errors"),
		CorrelationID: "1",
		ExternalID:    "1",
	}
}

func createServiceInstance(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	return &ServiceInstance{
		Base: Base{
			ID:        "id",
			Labels:    labels,
			CreatedAt: now,
			UpdatedAt: now.Add(time.Second * 10),
			Ready:     true,
		},
		Name:            "name",
		ServicePlanID:   "1",
		PlatformID:      "1",
		DashboardURL:    "http://test-service.com/dashboard",
		MaintenanceInfo: []byte("default"),
		Context:         []byte("default"),
		UpdateValues:    InstanceUpdateValues{},
		Usable:          true,
	}
}

func createNotification(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	return &Notification{
		Base: Base{
			ID:        "id",
			Labels:    labels,
			CreatedAt: now,
			UpdatedAt: now.Add(time.Second * 10),
		},
		Resource:      ObjectType("resource"),
		Type:          NotificationOperation("type"),
		PlatformID:    "1",
		Revision:      1,
		Payload:       []byte("payload"),
		CorrelationID: "1",
	}
}

func createServicePlan(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	trueVar := true
	return &ServicePlan{
		Base: Base{
			ID:             "id",
			CreatedAt:      now,
			UpdatedAt:      now.Add(time.Second * 10),
			Labels:         labels,
			PagingSequence: 0,
		},
		Name:              "name",
		Description:       "description",
		CatalogID:         "1",
		CatalogName:       "catname",
		Free:              &trueVar,
		Bindable:          &trueVar,
		PlanUpdatable:     &trueVar,
		Metadata:          []byte("metadata"),
		Schemas:           []byte("schema"),
		ServiceOfferingID: "1",
	}
}

func createServiceOffering(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	return &ServiceOffering{
		Base: Base{
			ID:             "id",
			CreatedAt:      now,
			UpdatedAt:      now.Add(time.Second * 10),
			Labels:         labels,
			PagingSequence: 0,
		},
		Name:                 "name",
		Description:          "description",
		Bindable:             true,
		InstancesRetrievable: true,
		BindingsRetrievable:  true,
		PlanUpdatable:        true,
		Tags:                 []byte("tags"),
		Requires:             []byte("requires"),
		Metadata:             []byte("metadata"),
		BrokerID:             "1",
		CatalogID:            "1",
		CatalogName:          "catname",
		Plans:                nil,
	}
}

func createVisibility(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	return &Visibility{
		Base: Base{
			ID:             "id",
			CreatedAt:      now,
			UpdatedAt:      now.Add(time.Second * 10),
			Labels:         labels,
			PagingSequence: 0,
		},
		PlatformID:    "1",
		ServicePlanID: "1",
	}
}

func createPlatform(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	return &Platform{
		Base: Base{
			ID:             "id",
			CreatedAt:      now,
			UpdatedAt:      now.Add(time.Second * 10),
			Labels:         labels,
			PagingSequence: 0,
		},
		Secured:     nil,
		Type:        "cloudfoundry",
		Name:        "test_broker",
		Description: "decription",
		Credentials: &Credentials{
			Basic: &Basic{
				Username: "user",
				Password: "password",
			},
		},
		Active:     true,
		LastActive: now,
	}
}

func createBroker(now time.Time) Object {
	labels := Labels{
		"label_key": []string{"value"},
	}
	return &ServiceBroker{
		Base: Base{
			ID:             "id",
			CreatedAt:      now,
			UpdatedAt:      now.Add(time.Second * 10),
			Labels:         labels,
			PagingSequence: 0,
		},
		Secured:     nil,
		Name:        "test_broker",
		Description: "broker decription",
		BrokerURL:   "broker_url",
		Credentials: &Credentials{
			Basic: &Basic{
				Username: "user",
				Password: "password",
			},
		},
		Catalog:  nil,
		Services: nil,
	}
}
