package handlers_test

import (
	"github.com/Peripli/service-manager/pkg/agent/notifications/handlers"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handlers Helpers", func() {
	Describe("LabelChangesToLabels", func() {
		Context("for changes with add and remove operations", func() {
			var labelChanges query.LabelChanges
			var expectedLabelsToAdd types.Labels
			var expectedLabelsToRemove types.Labels

			BeforeEach(func() {
				labelChanges = query.LabelChanges{
					&query.LabelChange{
						Operation: query.AddLabelOperation,
						Key:       "organization_guid",
						Values: []string{
							"org1",
							"org2",
						},
					},
					&query.LabelChange{
						Operation: query.AddLabelValuesOperation,
						Key:       "organization_guid",
						Values: []string{
							"org3",
							"org4",
						},
					},
					&query.LabelChange{
						Operation: query.RemoveLabelValuesOperation,
						Key:       "organization_guid",
						Values: []string{
							"org5",
							"org6",
						},
					},
					&query.LabelChange{
						Operation: query.RemoveLabelOperation,
						Key:       "organization_guid",
						Values: []string{
							"org7",
							"org8",
						},
					},
				}

				expectedLabelsToAdd = types.Labels{
					"organization_guid": {
						"org1",
						"org2",
						"org3",
						"org4",
					},
				}

				expectedLabelsToRemove = types.Labels{
					"organization_guid": {
						"org5",
						"org6",
						"org7",
						"org8",
					},
				}
			})

			It("generates correct labels", func() {
				labelsToAdd, labelsToRemove := handlers.LabelChangesToLabels(labelChanges)

				Expect(labelsToAdd).To(Equal(expectedLabelsToAdd))
				Expect(labelsToRemove).To(Equal(expectedLabelsToRemove))
			})
		})
	})
})
