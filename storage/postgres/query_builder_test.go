package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"

	"github.com/Peripli/service-manager/pkg/query"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/storage/postgres"

	"github.com/Peripli/service-manager/storage/postgres/postgresfakes"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
)

// The query builder tests contain the full queries that are expected to be build and can therefore be used as documentation
// to better understand the final queries that will be executed
var _ = Describe("Postgres Storage Query builder", func() {
	var executedQuery string
	var queryArgs []interface{}
	var ctx = context.Background()
	var entity *postgres.Visibility
	var qb *postgres.QueryBuilder

	trim := func(s string) string {
		return regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	}

	db := &postgresfakes.FakePgDB{}
	db.QueryxContextStub = func(ctx context.Context, query string, args ...interface{}) (rows *sqlx.Rows, e error) {
		executedQuery = query
		queryArgs = args
		return &sqlx.Rows{}, nil
	}
	db.SelectContextStub = func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
		executedQuery = query
		queryArgs = args
		return nil
	}
	db.ExecContextStub = func(ctx context.Context, query string, args ...interface{}) (result sql.Result, e error) {
		executedQuery = query
		queryArgs = args
		return driver.RowsAffected(1), nil
	}
	db.RebindStub = func(s string) string {
		return s
	}

	BeforeEach(func() {
		entity = &postgres.Visibility{}
		qb = postgres.NewQueryBuilder(db)
	})

	Describe("List", func() {
		Context("when no criteria is used", func() {
			It("builds simple query for entity and its labels", func() {
				_, err := qb.NewQuery().List(ctx, entity)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`SELECT t.*,
	visibility_labels.id "visibility_labels.id",
	visibility_labels.key "visibility_labels.key",
	visibility_labels.val "visibility_labels.val",
	visibility_labels.created_at "visibility_labels.created_at",
	visibility_labels.updated_at "visibility_labels.updated_at",
	visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id;`)))
				Expect(queryArgs).To(HaveLen(0))
			})
		})

		Context("when label criteria is used", func() {
			It("should build query with label criteria", func() {
				_, err := qb.NewQuery().
					WithCriteria(query.ByLabel(query.EqualsOperator, "labelKey", "labelValue")).
					List(ctx, entity)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`SELECT t.*,
	visibility_labels.id "visibility_labels.id",
	visibility_labels.key "visibility_labels.key",
	visibility_labels.val "visibility_labels.val",
	visibility_labels.created_at "visibility_labels.created_at",
	visibility_labels.updated_at "visibility_labels.updated_at",
	visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities t
JOIN
  (SELECT *
   FROM visibility_labels
   WHERE visibility_id IN
	   (SELECT visibility_id
		FROM visibility_labels
		WHERE (visibility_labels.key = ?
			   AND visibility_labels.val = ?))) visibility_labels ON t.id = visibility_labels.visibility_id;`)))
				Expect(queryArgs).To(HaveLen(2))
				Expect(queryArgs[0]).Should(Equal("labelKey"))
				Expect(queryArgs[1]).Should(Equal("labelValue"))
			})
		})

		Context("when field criteria is used", func() {
			It("builds query with field criteria", func() {
				_, err := qb.NewQuery().
					WithCriteria(query.ByField(query.EqualsOperator, "id", "1")).
					List(ctx, entity)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`SELECT t.*,
	visibility_labels.id "visibility_labels.id",
	visibility_labels.key "visibility_labels.key",
	visibility_labels.val "visibility_labels.val",
	visibility_labels.created_at "visibility_labels.created_at",
	visibility_labels.updated_at "visibility_labels.updated_at",
	visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id::text = ?;`)))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("1"))
			})

			Context("when field is missing", func() {
				It("returns error", func() {
					criteria := query.ByField(query.EqualsOperator, "non-existing-field", "value")
					_, err := qb.NewQuery().WithCriteria(criteria).List(ctx, entity)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when order by criteria is used", func() {
			It("builds query with order by clause", func() {
				_, err := qb.NewQuery().
					WithCriteria(query.OrderResultBy("id", query.DescOrder)).
					List(ctx, entity)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`SELECT t.*,
	visibility_labels.id "visibility_labels.id",
	visibility_labels.key "visibility_labels.key",
	visibility_labels.val "visibility_labels.val",
	visibility_labels.created_at "visibility_labels.created_at",
	visibility_labels.updated_at "visibility_labels.updated_at",
	visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id
ORDER BY t.id DESC;`)))
				Expect(queryArgs).To(HaveLen(0))
			})

			Context("when order type is unknown", func() {
				It("returns error", func() {
					_, err := qb.NewQuery().WithCriteria(query.OrderResultBy("id", "unknown-order")).List(ctx, entity)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("unsupported order type: unknown-order"))
				})
			})

			Context("when the field is unknown", func() {
				It("returns error", func() {
					_, err := qb.NewQuery().WithCriteria(query.OrderResultBy("unknown-field", query.AscOrder)).List(ctx, entity)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("unsupported entity field for order by: unknown-field"))
				})
			})

			Context("when order type is missing", func() {
				It("returns error", func() {
					_, err := qb.NewQuery().
						WithCriteria(query.Criterion{
							Type:    query.ResultQuery,
							LeftOp:  query.OrderBy,
							RightOp: []string{"id"},
						}).
						List(ctx, entity)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(`order by result for field "id" expects order type, but has none`))
				})
			})

			Context("when order type and field are missing", func() {
				It("return errors", func() {
					_, err := qb.NewQuery().
						WithCriteria(query.Criterion{
							Type:    query.ResultQuery,
							LeftOp:  query.OrderBy,
							RightOp: []string{},
						}).
						List(ctx, entity)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("order by result expects field and order type, but has none"))
				})
			})
		})

		Context("when limit criteria is used", func() {
			It("builds query with limit clause", func() {
				_, err := qb.NewQuery().
					WithCriteria(query.LimitResultBy(10)).
					List(ctx, entity)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`SELECT t.*,
	visibility_labels.id "visibility_labels.id",
	visibility_labels.key "visibility_labels.key",
	visibility_labels.val "visibility_labels.val",
	visibility_labels.created_at "visibility_labels.created_at",
	visibility_labels.updated_at "visibility_labels.updated_at",
	visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id
LIMIT 10;`)))
				Expect(queryArgs).To(HaveLen(0))
			})

			Context("when limit is negative", func() {
				It("returns error", func() {
					_, err := qb.NewQuery().WithCriteria(query.LimitResultBy(-1)).List(ctx, entity)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("limit (-1) is invalid. Limit should be positive number"))
				})
			})
		})

		Context("when multiple criteria are used", func() {
			It("builds a valid query", func() {
				criteria1 := query.ByField(query.NotEqualsOperator, "id", "1")
				criteria2 := query.ByField(query.NotInOperator, "service_plan_id", "2", "3", "4")
				criteria3 := query.ByField(query.EqualsOrNilOperator, "platform_id", "5")

				criteria4 := query.ByLabel(query.EqualsOperator, "left1", "right1")
				criteria5 := query.ByLabel(query.InOperator, "left2", "right2", "right3")
				criteria6 := query.ByLabel(query.NotEqualsOperator, "left3", "right4")

				_, err := qb.NewQuery().
					WithCriteria(criteria1, criteria2, criteria3, criteria4, criteria5, criteria6).
					List(ctx, entity)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`SELECT t.*,
	visibility_labels.id "visibility_labels.id",
	visibility_labels.key "visibility_labels.key",
	visibility_labels.val "visibility_labels.val",
	visibility_labels.created_at "visibility_labels.created_at",
	visibility_labels.updated_at "visibility_labels.updated_at",
	visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities t
JOIN
  (SELECT *
   FROM visibility_labels
   WHERE visibility_id IN
	   (SELECT visibility_id
		FROM visibility_labels
		WHERE (visibility_labels.key = ?
			   AND visibility_labels.val = ?)
		  OR (visibility_labels.key = ?
			  AND visibility_labels.val IN (?, ?))
		  OR (visibility_labels.key = ?
			  AND visibility_labels.val != ?))) visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id::text != ?
  AND t.service_plan_id::text NOT IN (?, ?, ?)
  AND (t.platform_id::text = ?
	   OR t.platform_id IS NULL);`)))
				Expect(queryArgs).To(HaveLen(12))
				Expect(queryArgs[0]).Should(Equal("left1"))
				Expect(queryArgs[1]).Should(Equal("right1"))
				Expect(queryArgs[2]).Should(Equal("left2"))
				Expect(queryArgs[3]).Should(Equal("right2"))
				Expect(queryArgs[4]).Should(Equal("right3"))
				Expect(queryArgs[5]).Should(Equal("left3"))
				Expect(queryArgs[6]).Should(Equal("right4"))
				Expect(queryArgs[7]).Should(Equal("1"))
				Expect(queryArgs[8]).Should(Equal("2"))
				Expect(queryArgs[9]).Should(Equal("3"))
				Expect(queryArgs[10]).Should(Equal("4"))
				Expect(queryArgs[11]).Should(Equal("5"))
			})
		})
	})

	Describe("Delete", func() {
		Context("when no criteria is used", func() {
			It("builds query to delete all entries", func() {
				_, err := qb.NewQuery().Delete(ctx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(executedQuery).To(Equal(trim(`DELETE
FROM visibilities USING visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id = visibilities.id;`)))
			})
		})

		Context("when label criteria is used", func() {
			It("builds query with label criteria", func() {
				criteria1 := query.ByLabel(query.EqualsOperator, "left1", "right1")
				criteria2 := query.ByLabel(query.InOperator, "left2", "right2", "right3")
				_, err := qb.NewQuery().WithCriteria(criteria1, criteria2).Delete(ctx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`DELETE
FROM visibilities USING visibilities t
JOIN
  (SELECT *
   FROM visibility_labels
   WHERE visibility_id IN
	   (SELECT visibility_id
		FROM visibility_labels
		WHERE (visibility_labels.key = ?
			   AND visibility_labels.val = ?)
		  OR (visibility_labels.key = ?
			  AND visibility_labels.val IN (?, ?)))) visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id = visibilities.id;`)))
				Expect(queryArgs).To(HaveLen(5))
				Expect(queryArgs[0]).Should(Equal("left1"))
				Expect(queryArgs[1]).Should(Equal("right1"))
				Expect(queryArgs[2]).Should(Equal("left2"))
				Expect(queryArgs[3]).Should(Equal("right2"))
				Expect(queryArgs[4]).Should(Equal("right3"))
			})
		})

		Context("when field criteria is used", func() {
			It("builds query with field criteria", func() {
				criteria := query.ByField(query.EqualsOperator, "id", "1")
				_, err := qb.NewQuery().WithCriteria(criteria).Delete(ctx, entity)
				Expect(err).ToNot(HaveOccurred())

				Expect(executedQuery).Should(Equal(trim(`DELETE
FROM visibilities USING visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id = visibilities.id
  AND t.id::text = ?;`)))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("1"))
			})

			Context("when field is missing", func() {
				It("returns error", func() {
					criteria := query.ByField(query.EqualsOperator, "non-existing-field", "value")
					_, err := qb.NewQuery().WithCriteria(criteria).Delete(ctx, entity)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when returning clause is used", func() {
			It("builds query with specified fields fields", func() {
				_, err := qb.NewQuery().Return("id", "service_plan_id").Delete(ctx, entity)
				Expect(err).ToNot(HaveOccurred())

				Expect(executedQuery).To(Equal(trim(`DELETE
FROM visibilities USING visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id = visibilities.id 
RETURNING t.id, t.service_plan_id;`)))
			})

			It("builds query with *", func() {
				_, err := qb.NewQuery().Return("*").Delete(ctx, entity)
				Expect(err).ToNot(HaveOccurred())

				Expect(executedQuery).To(Equal(trim(`DELETE
FROM visibilities USING visibilities t
LEFT JOIN visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id = visibilities.id 
RETURNING t.*;`)))
			})

			Context("when unknown field is specified", func() {
				It("returns error", func() {
					_, err := qb.NewQuery().Return("unknown-field").Delete(ctx, entity)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("unsupported entity field for return type: unknown-field"))
				})
			})
		})

		Context("when multiple criteria are used", func() {
			It("builds a valid query", func() {
				criteria1 := query.ByField(query.NotEqualsOperator, "id", "1")
				criteria2 := query.ByField(query.NotInOperator, "service_plan_id", "2", "3", "4")
				criteria3 := query.ByField(query.EqualsOrNilOperator, "platform_id", "5")

				criteria4 := query.ByLabel(query.EqualsOperator, "left1", "right1")
				criteria5 := query.ByLabel(query.InOperator, "left2", "right2", "right3")
				criteria6 := query.ByLabel(query.NotEqualsOperator, "left3", "right4")

				_, err := qb.NewQuery().WithCriteria(criteria1, criteria2, criteria3, criteria4, criteria5, criteria6).Return("*").Delete(ctx, entity)
				Expect(err).ToNot(HaveOccurred())

				Expect(executedQuery).Should(Equal(trim(`DELETE
FROM visibilities USING visibilities t
JOIN
  (SELECT *
   FROM visibility_labels
   WHERE visibility_id IN
	   (SELECT visibility_id
		FROM visibility_labels
		WHERE (visibility_labels.key = ?
			   AND visibility_labels.val = ?)
		  OR (visibility_labels.key = ?
			  AND visibility_labels.val IN (?, ?))
		  OR (visibility_labels.key = ?
			  AND visibility_labels.val != ?))) visibility_labels ON t.id = visibility_labels.visibility_id
WHERE t.id = visibilities.id
  AND t.id::text != ?
  AND t.service_plan_id::text NOT IN (?, ?, ?)
  AND (t.platform_id::text = ?
	   OR t.platform_id IS NULL) 
RETURNING t.*;`)))
				Expect(queryArgs).To(HaveLen(12))
				Expect(queryArgs[0]).Should(Equal("left1"))
				Expect(queryArgs[1]).Should(Equal("right1"))
				Expect(queryArgs[2]).Should(Equal("left2"))
				Expect(queryArgs[3]).Should(Equal("right2"))
				Expect(queryArgs[4]).Should(Equal("right3"))
				Expect(queryArgs[5]).Should(Equal("left3"))
				Expect(queryArgs[6]).Should(Equal("right4"))
				Expect(queryArgs[7]).Should(Equal("1"))
				Expect(queryArgs[8]).Should(Equal("2"))
				Expect(queryArgs[9]).Should(Equal("3"))
				Expect(queryArgs[10]).Should(Equal("4"))
				Expect(queryArgs[11]).Should(Equal("5"))
			})
		})
	})
})
