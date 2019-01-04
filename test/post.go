package test

import (
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

//Context("With invalid content type", func() {
//	It("returns 415", func() {
//		ctx.SMWithOAuth.POST("/v1/platforms").WithHeader().WithBytes()
//		WithText("text").
//			Expect().Status(http.StatusUnsupportedMediaType)

//Context("With invalid content JSON", func() {
//	It("returns 400 if input is not valid JSON", func() {
//		ctx.SMWithOAuth.POST("/v1/platforms").
//			WithText("invalid json").
//			WithHeader("content-type", "application/json").

//Context("With missing mandatory fields", func() {
//	It("returns 400", func() {

//Context("with already existing resource", func() {
//	It("returns 409", func() {

//Context("With optional fields skipped", func() {
//	It("succeeds", func() {

//Context("With invalid id", func() {
//	It("fails", func() {//id:="platform/,<1"

//Context("Without id", func() {
//	It("returns the new platform with generated id and credentials", func() {

// Context("when request body is missing")

// for brokers
// when broker is not reachable
// when broker is missing
// when broker is failing
// when broker is working
	//with invalid catalog
		// when services/plans  with same id exist in catalog
		// when services/plans with same name exist in catalog
		// when a service/plan is incomplete / missing mandatory fields
	// with valid catalog
		// when service/plan is incomplete/ missing optional fields
		--// when catalog is complete
		// when broker url ends with trailing slash
		// when broker url does not end with trailing slash

func DescribePostTestsFor(ctx *common.TestContext, t TestCase) bool {
	return Describe("POST", func() {

	})
}
