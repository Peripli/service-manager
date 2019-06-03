package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/query"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/storage/postgres"

	"github.com/Peripli/service-manager/storage/postgres/postgresfakes"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Postgres Storage Query builder", func() {
	var executedQuery string
	var queryArgs []interface{}

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

	Describe("List", func() {
		Context("when there are no criterias", func() {
			It("should build simple query for labeable entity", func() {
				qb := postgres.NewQueryBuilder(db)
				_, err := qb.List(context.Background(), &postgres.Visibility{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities LEFT JOIN"))
				Expect(queryArgs).To(HaveLen(0))
			})
		})

		Context("when field criteria is used", func() {
			It("should build right query", func() {
				qb := postgres.NewQueryBuilder(db)
				_, err := qb.
					WithCriteria(query.ByField(query.EqualsOperator, "id", "1")).
					List(context.Background(), &postgres.Visibility{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities LEFT JOIN .* WHERE"))
				Expect(queryArgs).To(HaveLen(1))
				Expect(queryArgs[0]).Should(Equal("1"))
			})
		})

		Context("when label criteria is used", func() {
			It("should build query with label criteria", func() {
				qb := postgres.NewQueryBuilder(db)
				_, err := qb.
					WithCriteria(query.ByLabel(query.EqualsOperator, "labelKey", "labelValue")).
					List(context.Background(), &postgres.Visibility{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities JOIN \\(SELECT.*\\)"))
				Expect(queryArgs).To(HaveLen(2))
				Expect(queryArgs[0]).Should(Equal("labelKey"))
				Expect(queryArgs[1]).Should(Equal("labelValue"))
			})
		})

		Context("when list criteria is used", func() {
			It("should build query with order by clause", func() {
				qb := postgres.NewQueryBuilder(db)
				_, err := qb.
					WithListCriteria(storage.ListCriteria{
						Type:      storage.OrderByCriteriaType,
						Parameter: "id",
					}).
					List(context.Background(), &postgres.Visibility{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities .* ORDER BY id;"))
				Expect(queryArgs).To(HaveLen(0))
			})

			It("should build query with limit clause", func() {
				qb := postgres.NewQueryBuilder(db)
				_, err := qb.
					WithListCriteria(storage.ListCriteria{
						Type:      storage.LimitCriteriaType,
						Parameter: 10,
					}).
					List(context.Background(), &postgres.Visibility{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities .* LIMIT 10;"))
				Expect(queryArgs).To(HaveLen(0))
			})

			It("should build query with order by and limit clause", func() {
				qb := postgres.NewQueryBuilder(db)
				_, err := qb.
					WithListCriteria(storage.ListCriteria{
						Type:      storage.LimitCriteriaType,
						Parameter: 10,
					}, storage.ListCriteria{
						Type:      storage.OrderByCriteriaType,
						Parameter: "id",
					}).
					List(context.Background(), &postgres.Visibility{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities .* ORDER BY id LIMIT 10;"))
				Expect(queryArgs).To(HaveLen(0))
			})
		})
	})

	Describe("Delete", func() {
		Context("When deleting by label", func() {
			It("Should return an error", func() {
				qb := postgres.NewQueryBuilder(db)
				criteria := query.ByLabel(query.EqualsOperator, "left", "right")
				_, err := qb.WithCriteria(criteria).Delete(context.Background(), &postgres.Visibility{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When no criteria is passed", func() {
			It("Should construct query to delete all entries", func() {
				qb := postgres.NewQueryBuilder(db)
				_, err := qb.Return("*").Delete(context.Background(), &postgres.Visibility{})
				expectedQuery := fmt.Sprintf("DELETE FROM visibilities RETURNING *;")
				Expect(err).ToNot(HaveOccurred())
				Expect(executedQuery).To(Equal(expectedQuery))
			})
		})

		Context("When criteria uses missing field", func() {
			It("Should return error", func() {
				qb := postgres.NewQueryBuilder(db)
				criteria := query.ByField(query.EqualsOperator, "non-existing-field", "value")
				_, err := qb.WithCriteria(criteria).Delete(context.Background(), &postgres.Visibility{})
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
