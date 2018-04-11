package osb_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOsb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Osb Suite")
}
