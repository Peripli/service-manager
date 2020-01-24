/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package profile_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
)

func TestProfile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Profile Suite")
}

var _ = Describe("Profile API", func() {

	var ctx *common.TestContext
	var smURL, tempDir string

	BeforeSuite(func() {
		ctx = common.DefaultTestContext()
		smURL = ctx.Servers[common.SMServer].URL()
		var err error
		tempDir, err = ioutil.TempDir("", "pprof")
		Expect(err).To(BeNil())
	})

	AfterSuite(func() {
		ctx.Cleanup()
		os.RemoveAll(tempDir)
	})

	Describe("Get unknown profile", func() {
		It("Returns 404 response", func() {
			ctx.SM.GET(web.ProfileURL + "/unknown").
				Expect().
				Status(http.StatusNotFound)
		})
	})

	Describe("pprof", func() {
		profiles := []string{
			"goroutine",
			"threadcreate",
			"heap",
			"allocs",
			"block",
			"mutex",
		}
		for _, name := range profiles {
			name := name
			It("accepts "+name+" profile", func() {
				profileURL := smURL + web.ProfileURL + "/" + name
				cmd := exec.Command("go", "tool", "pprof", "-top", profileURL)
				cmd.Env = append(os.Environ(), "PPROF_TMPDIR="+tempDir)
				cmd.Stdout = ginkgo.GinkgoWriter
				cmd.Stderr = ginkgo.GinkgoWriter
				common.Print("%s %s", cmd.Path, strings.Join(cmd.Args[1:], " "))
				err := cmd.Run()
				Expect(err).To(BeNil())
			})
		}
	})
})
