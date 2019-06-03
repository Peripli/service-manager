package postgres_test

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/Peripli/service-manager/pkg/query"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/storage/postgres"

	"github.com/Peripli/service-manager/storage/postgres/postgresfakes"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Postgres Storage Abstract", func() {
	// var ctx context.Context
	// var baseTable string
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

	Context("when there are no criterias", func() {
		It("should build simple query for labeable entity", func() {
			qb := postgres.NewQueryBuilder(db, &postgres.Visibility{})
			_, err := qb.List(context.Background())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities LEFT JOIN"))
			Expect(queryArgs).To(HaveLen(0))
		})
	})

	Context("when field criteria is used", func() {
		It("should build right query", func() {
			qb := postgres.NewQueryBuilder(db, &postgres.Visibility{})
			_, err := qb.
				WithCriteria(query.ByField(query.EqualsOperator, "id", "1")).
				List(context.Background())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities LEFT JOIN.*WHERE"))
			Expect(queryArgs).To(HaveLen(1))
			Expect(queryArgs[0]).Should(Equal("1"))
		})
	})

	Context("when label criteria is used", func() {
		It("should build query with label criteria", func() {
			qb := postgres.NewQueryBuilder(db, &postgres.Visibility{})
			_, err := qb.
				WithCriteria(query.ByLabel(query.EqualsOperator, "labelKey", "labelValue")).
				List(context.Background())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(executedQuery).Should(MatchRegexp("SELECT.*FROM visibilities JOIN \\(SELECT.*\\)"))
			Expect(queryArgs).To(HaveLen(2))
			Expect(queryArgs[0]).Should(Equal("labelKey"))
			Expect(queryArgs[1]).Should(Equal("labelValue"))
		})
	})
})
