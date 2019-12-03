package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"strings"

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
		return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
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
	db.GetContextStub = func(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
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
		Context("when entity does not have an associated label entity", func() {
			It("returns error", func() {
				_, err := qb.NewQuery(&postgres.Safe{}).List(ctx)
				Expect(err.Error()).To(ContainSubstring("query builder requires the entity to have associated label entity"))
			})
		})

		Context("when no criteria is used", func() {
			It("builds simple query for entity and its labels", func() {
				_, err := qb.NewQuery(entity).List(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT visibilities.*,
       visibility_labels.id            "visibility_labels.id",
       visibility_labels.key           "visibility_labels.key",
       visibility_labels.val           "visibility_labels.val",
       visibility_labels.created_at    "visibility_labels.created_at",
       visibility_labels.updated_at    "visibility_labels.updated_at",
       visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities
         LEFT JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id ;`)))
				Expect(queryArgs).To(HaveLen(0))
			})
		})

		Context("when label criteria is used", func() {
			It("should build query with label criteria", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.ByLabel(query.EqualsOperator, "labelKey", "labelValue")).
					List(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
WITH matching_resources as (SELECT DISTINCT visibilities.paging_sequence
                            FROM visibilities
								JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
                            WHERE (key::text = ? AND val::text = ?) )
SELECT visibilities.*,
       visibility_labels.id            "visibility_labels.id",
       visibility_labels.key           "visibility_labels.key",
       visibility_labels.val           "visibility_labels.val",
       visibility_labels.created_at    "visibility_labels.created_at",
       visibility_labels.updated_at    "visibility_labels.updated_at",
       visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities
	JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
WHERE visibilities.paging_sequence IN (SELECT matching_resources.paging_sequence FROM matching_resources) ;`)))
				Expect(queryArgs).To(HaveLen(2))
				Expect(queryArgs[0]).Should(Equal("labelKey"))
				Expect(queryArgs[1]).Should(Equal("labelValue"))
			})
		})

		Context("when field criteria is used", func() {
			It("builds query with field criteria", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.ByField(query.EqualsOperator, "id", "1")).
					List(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
WITH matching_resources as (SELECT DISTINCT visibilities.paging_sequence
                            FROM visibilities
                            WHERE visibilities.id::text = ? )
SELECT visibilities.*,
       visibility_labels.id            "visibility_labels.id",
       visibility_labels.key           "visibility_labels.key",
       visibility_labels.val           "visibility_labels.val",
       visibility_labels.created_at    "visibility_labels.created_at",
       visibility_labels.updated_at    "visibility_labels.updated_at",
       visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities
         LEFT JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
WHERE visibilities.paging_sequence IN (SELECT matching_resources.paging_sequence FROM matching_resources) ;`)))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("1"))
			})

			Context("when field is missing", func() {
				It("returns error", func() {
					criteria := query.ByField(query.EqualsOperator, "non-existing-field", "value")
					_, err := qb.NewQuery(entity).WithCriteria(criteria).List(ctx)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when order by criteria is used", func() {
			It("builds query with order by clause", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.OrderResultBy("id", query.DescOrder), query.OrderResultBy("created_at", query.AscOrder)).
					List(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT visibilities.*,
       visibility_labels.id            "visibility_labels.id",
       visibility_labels.key           "visibility_labels.key",
       visibility_labels.val           "visibility_labels.val",
       visibility_labels.created_at    "visibility_labels.created_at",
       visibility_labels.updated_at    "visibility_labels.updated_at",
       visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities
         LEFT JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
ORDER BY id DESC, created_at ASC ;`)))
				Expect(queryArgs).To(HaveLen(0))
			})

			Context("when order type is unknown", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).WithCriteria(query.OrderResultBy("id", "unknown-order")).List(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("unsupported order type: unknown-order"))
				})
			})

			Context("when the field is unknown", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).WithCriteria(query.OrderResultBy("unknown-field", query.AscOrder)).List(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("unsupported entity field for order by: unknown-field"))
				})
			})

			Context("when order type is missing", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).
						WithCriteria(query.Criterion{
							Type:    query.ResultQuery,
							LeftOp:  query.OrderBy,
							RightOp: []string{"id"},
						}).
						List(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("order by result expects field name and order type"))
				})
			})

			Context("when order type and field are missing", func() {
				It("return errors", func() {
					_, err := qb.NewQuery(entity).
						WithCriteria(query.Criterion{
							Type:    query.ResultQuery,
							LeftOp:  query.OrderBy,
							RightOp: []string{},
						}).
						List(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("missing right operand"))
				})
			})
		})

		Context("when limit criteria is used", func() {
			It("builds query with limit clause", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.LimitResultBy(10)).
					List(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
WITH matching_resources as (SELECT DISTINCT visibilities.paging_sequence
                            FROM visibilities
							ORDER BY visibilities.paging_sequence ASC
                            LIMIT ?)
SELECT visibilities.*,
       visibility_labels.id            "visibility_labels.id",
       visibility_labels.key           "visibility_labels.key",
       visibility_labels.val           "visibility_labels.val",
       visibility_labels.created_at    "visibility_labels.created_at",
       visibility_labels.updated_at    "visibility_labels.updated_at",
       visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities
         LEFT JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
WHERE visibilities.paging_sequence IN (SELECT matching_resources.paging_sequence FROM matching_resources) ;`)))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("10"))
			})

			Context("when limit is negative", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).WithCriteria(query.LimitResultBy(-1)).List(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("limit (-1) is invalid. Limit should be positive number"))
				})
			})
			Context("when multiple limit criteria is used", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).WithCriteria(query.LimitResultBy(10)).
						WithCriteria(query.LimitResultBy(5)).List(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("zero/one limit expected but multiple provided"))
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

				criteria7 := query.LimitResultBy(10)
				criteria8 := query.OrderResultBy("id", query.AscOrder)

				_, err := qb.NewQuery(entity).
					WithCriteria(criteria1, criteria2, criteria3, criteria4, criteria5, criteria6, criteria7, criteria8).
					List(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
WITH matching_resources as (SELECT DISTINCT visibilities.paging_sequence
                            FROM visibilities
								JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
                            WHERE ((visibilities.id::text != ? AND
                                    visibilities.service_plan_id::text NOT IN (?, ?, ?) AND
                                    (visibilities.platform_id::text = ? OR platform_id IS NULL)) AND
                                   ((key::text = ? AND val::text = ?) OR (key::text = ? AND val::text IN (?, ?)) OR
                                    (key::text = ? AND val::text != ?)))
                            ORDER BY visibilities.paging_sequence ASC
                            LIMIT ?)
SELECT visibilities.*,
       visibility_labels.id            "visibility_labels.id",
       visibility_labels.key           "visibility_labels.key",
       visibility_labels.val           "visibility_labels.val",
       visibility_labels.created_at    "visibility_labels.created_at",
       visibility_labels.updated_at    "visibility_labels.updated_at",
       visibility_labels.visibility_id "visibility_labels.visibility_id"
FROM visibilities
	JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
WHERE visibilities.paging_sequence IN (SELECT matching_resources.paging_sequence FROM matching_resources)
ORDER BY id ASC ;`)))
				Expect(queryArgs).To(HaveLen(13))
				Expect(queryArgs[0]).Should(Equal("1"))
				Expect(queryArgs[1]).Should(Equal("2"))
				Expect(queryArgs[2]).Should(Equal("3"))
				Expect(queryArgs[3]).Should(Equal("4"))
				Expect(queryArgs[4]).Should(Equal("5"))
				Expect(queryArgs[5]).Should(Equal("left1"))
				Expect(queryArgs[6]).Should(Equal("right1"))
				Expect(queryArgs[7]).Should(Equal("left2"))
				Expect(queryArgs[8]).Should(Equal("right2"))
				Expect(queryArgs[9]).Should(Equal("right3"))
				Expect(queryArgs[10]).Should(Equal("left3"))
				Expect(queryArgs[11]).Should(Equal("right4"))
				Expect(queryArgs[12]).Should(Equal("10"))
			})
		})
	})

	Describe("Count", func() {
		Context("when entity does not have an associated label entity", func() {
			It("returns error", func() {
				_, err := qb.NewQuery(&postgres.Safe{}).Count(ctx)
				Expect(err.Error()).To(ContainSubstring("query builder requires the entity to have associated label entity"))
			})
		})

		Context("when no criteria is used", func() {
			It("builds simple query for entity and its labels", func() {
				_, err := qb.NewQuery(entity).Count(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT COUNT(DISTINCT visibilities.id)
FROM visibilities ;`)))
				Expect(queryArgs).To(HaveLen(0))
			})
		})

		Context("when label criteria is used", func() {
			It("should build query with label criteria", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.ByLabel(query.EqualsOperator, "labelKey", "labelValue")).
					Count(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT COUNT(DISTINCT visibilities.id)
FROM visibilities
	JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
WHERE (key::text = ? AND val::text = ?) ;`)))
				Expect(queryArgs).To(HaveLen(2))
				Expect(queryArgs[0]).Should(Equal("labelKey"))
				Expect(queryArgs[1]).Should(Equal("labelValue"))
			})
		})

		Context("when field criteria is used", func() {
			It("builds query with field criteria", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.ByField(query.EqualsOperator, "id", "1")).
					Count(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT COUNT(DISTINCT visibilities.id)
FROM visibilities
WHERE visibilities.id::text = ? ;`)))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("1"))
			})

			Context("when field is missing", func() {
				It("returns error", func() {
					criteria := query.ByField(query.EqualsOperator, "non-existing-field", "value")
					_, err := qb.NewQuery(entity).WithCriteria(criteria).Count(ctx)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when order by criteria is used", func() {
			It("skips order", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.OrderResultBy("id", query.DescOrder)).
					Count(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT COUNT(DISTINCT visibilities.id)
FROM visibilities ;`)))
				Expect(queryArgs).To(HaveLen(0))
			})
		})

		Context("when limit criteria is used", func() {
			It("builds query with limit clause", func() {
				_, err := qb.NewQuery(entity).
					WithCriteria(query.LimitResultBy(10)).
					Count(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT COUNT(DISTINCT visibilities.id)
FROM visibilities
LIMIT ?;`)))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("10"))
			})

			Context("when limit is negative", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).WithCriteria(query.LimitResultBy(-1)).Count(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("limit (-1) is invalid. Limit should be positive number"))
				})
			})

			Context("when multiple limit criteria is used", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).WithCriteria(query.LimitResultBy(10)).
						WithCriteria(query.LimitResultBy(5)).Count(ctx)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("zero/one limit expected but multiple provided"))
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

				criteria7 := query.LimitResultBy(10)
				criteria8 := query.OrderResultBy("id", query.AscOrder)

				_, err := qb.NewQuery(entity).
					WithCriteria(criteria1, criteria2, criteria3, criteria4, criteria5, criteria6, criteria7, criteria8).
					Count(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
SELECT COUNT(DISTINCT visibilities.id)
FROM visibilities
         JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
WHERE ((visibilities.id::text != ? AND
	visibilities.service_plan_id::text NOT IN (?, ?, ?) AND
		(visibilities.platform_id::text = ? OR platform_id IS NULL)) AND
		((key::text = ? AND val::text = ?) OR (key::text = ? AND val::text IN (?, ?)) OR
		(key::text = ? AND val::text != ?)))
LIMIT ?;`)))
				Expect(queryArgs).To(HaveLen(13))
				Expect(queryArgs[0]).Should(Equal("1"))
				Expect(queryArgs[1]).Should(Equal("2"))
				Expect(queryArgs[2]).Should(Equal("3"))
				Expect(queryArgs[3]).Should(Equal("4"))
				Expect(queryArgs[4]).Should(Equal("5"))
				Expect(queryArgs[5]).Should(Equal("left1"))
				Expect(queryArgs[6]).Should(Equal("right1"))
				Expect(queryArgs[7]).Should(Equal("left2"))
				Expect(queryArgs[8]).Should(Equal("right2"))
				Expect(queryArgs[9]).Should(Equal("right3"))
				Expect(queryArgs[10]).Should(Equal("left3"))
				Expect(queryArgs[11]).Should(Equal("right4"))
				Expect(queryArgs[12]).Should(Equal("10"))
			})
		})
	})

	Describe("Delete", func() {
		Context("when entity does not have an associated label entity", func() {
			It("returns error", func() {
				_, err := qb.NewQuery(&postgres.Safe{}).Delete(ctx)
				Expect(err.Error()).To(ContainSubstring("query builder requires the entity to have associated label entity"))
			})
		})

		Context("when no criteria is used", func() {
			It("builds query to delete all entries", func() {
				_, err := qb.NewQuery(entity).Delete(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(executedQuery).To(Equal(trim(`
DELETE
FROM visibilities ;`)))
			})
		})

		Context("when label criteria is used", func() {
			It("builds query with label criteria", func() {
				criteria1 := query.ByLabel(query.EqualsOperator, "left1", "right1")
				criteria2 := query.ByLabel(query.InOperator, "left2", "right2", "right3")
				_, err := qb.NewQuery(entity).WithCriteria(criteria1, criteria2).Delete(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
DELETE
FROM visibilities USING (SELECT visibilities.id
                         FROM visibilities
                                  JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
                         WHERE ((key::text = ? AND val::text = ?) OR (key::text = ? AND val::text IN (?, ?)))) t
WHERE visibilities.id = t.id ;`)))
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
				_, err := qb.NewQuery(entity).WithCriteria(criteria).Delete(ctx)
				Expect(err).ToNot(HaveOccurred())

				Expect(executedQuery).Should(Equal(trim(`
DELETE
FROM visibilities
WHERE visibilities.id::text = ? ;`)))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("1"))
			})

			Context("when field is missing", func() {
				It("returns error", func() {
					criteria := query.ByField(query.EqualsOperator, "non-existing-field", "value")
					_, err := qb.NewQuery(entity).WithCriteria(criteria).Delete(ctx)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when returning clause is used", func() {
			It("builds query with specified fields fields", func() {
				_, err := qb.NewQuery(entity).Return("id", "service_plan_id").Delete(ctx)
				Expect(err).ToNot(HaveOccurred())

				Expect(executedQuery).To(Equal(trim(`
DELETE
FROM visibilities
RETURNING id, service_plan_id;`)))
			})

			It("builds query with *", func() {
				_, err := qb.NewQuery(entity).Return("*").Delete(ctx)
				Expect(err).ToNot(HaveOccurred())

				Expect(executedQuery).To(Equal(trim(`
DELETE
FROM visibilities
RETURNING *;`)))
			})

			Context("when unknown field is specified", func() {
				It("returns error", func() {
					_, err := qb.NewQuery(entity).Return("unknown-field").Delete(ctx)
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

				criteria7 := query.OrderResultBy("id", query.AscOrder)

				_, err := qb.NewQuery(entity).WithCriteria(criteria1, criteria2, criteria3, criteria4, criteria5, criteria6, criteria7).Return("*").Delete(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(executedQuery).Should(Equal(trim(`
DELETE
FROM visibilities USING (SELECT visibilities.id
                         FROM visibilities
                                  JOIN visibility_labels ON visibilities.id = visibility_labels.visibility_id
                         WHERE ((visibilities.id::text != ? AND visibilities.service_plan_id::text NOT IN (?, ?, ?) AND
                                 (visibilities.platform_id::text = ? OR platform_id IS NULL)) AND
                                ((key::text = ? AND val::text = ?) OR (key::text = ? AND val::text IN (?, ?)) OR
                                 (key::text = ? AND val::text != ?)))) t
WHERE visibilities.id = t.id RETURNING *;`)))
				Expect(queryArgs).To(HaveLen(12))
				Expect(queryArgs[0]).Should(Equal("1"))
				Expect(queryArgs[1]).Should(Equal("2"))
				Expect(queryArgs[2]).Should(Equal("3"))
				Expect(queryArgs[3]).Should(Equal("4"))
				Expect(queryArgs[4]).Should(Equal("5"))
				Expect(queryArgs[5]).Should(Equal("left1"))
				Expect(queryArgs[6]).Should(Equal("right1"))
				Expect(queryArgs[7]).Should(Equal("left2"))
				Expect(queryArgs[8]).Should(Equal("right2"))
				Expect(queryArgs[9]).Should(Equal("right3"))
				Expect(queryArgs[10]).Should(Equal("left3"))
				Expect(queryArgs[11]).Should(Equal("right4"))
			})
		})
	})
})
