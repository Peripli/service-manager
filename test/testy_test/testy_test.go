package testy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestTesty(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testy Tests Suite")

}

var _ = Describe("Bar", func() {
	entries := []TableEntry{
		FEntry("yolo1", 0),
		Entry("yolo2", 0),
		Entry("yolo3", 1),
		Entry("yolo4", 0),
		Entry("yolo5", 0),
	}

	testFunc := func(i int) {
		Expect(i).To(Equal(0))
	}

	FDescribeTable("table",
		testFunc, entries...,
	)

	DescribeTable("table",
		testFunc, entries...,
	)
})
