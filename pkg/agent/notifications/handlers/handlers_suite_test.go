package handlers_test

import (
	"fmt"
	"testing"

	"github.com/Peripli/service-manager/test/testutil"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handlers Suite")
}

func VerifyErrorLogged(f func()) {
	hook := &testutil.LogInterceptor{}
	logrus.AddHook(hook)
	Expect(hook).ToNot(ContainSubstring("error"))
	f()
	Expect(hook).To(ContainSubstring("error"))
}

func brokerProxyName(prefix, brokerName, brokerID string) string {
	return fmt.Sprintf("%s%s-%s", prefix, brokerName, brokerID)
}
