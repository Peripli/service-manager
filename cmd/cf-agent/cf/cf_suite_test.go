package cf_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCF(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Broker Proxy CF Client Suite")
}
