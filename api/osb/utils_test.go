package osb

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OSB Controller Utils test", func() {
	Describe("marshalJSONNoHTMLEscape", func() {
		It("keeps special characters", func() {
			inputMap := map[string]string{"prop": "a & b"}
			expected := []byte(`{"prop":"a & b"}`)
			notExpected := []byte(`{"prop":"a \u0026 b"}`)

			marshalNoEscapeBytes, err := marshalJSONNoHTMLEscape(inputMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(marshalNoEscapeBytes).To(Equal(expected))

			marshalBytes, err := json.Marshal(inputMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(marshalBytes).To(Equal(notExpected))

			Expect(marshalNoEscapeBytes).ToNot(Equal(marshalBytes))
		})

		It("eliminates line break added in the end", func() {
			inputMap := map[string]string{"prop": "val"}
			expected := []byte(`{"prop":"val"}`)

			marshalNoEscapeBytes, err := marshalJSONNoHTMLEscape(inputMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(marshalNoEscapeBytes).To(Equal(expected))

			marshalBytes, err := json.Marshal(inputMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(marshalBytes).To(Equal(expected))

			Expect(marshalNoEscapeBytes).To(Equal(marshalBytes))
		})

		It("returns empty byte array properly", func() {
			inputMap := map[string]string{}
			expected := []byte(`{}`)

			marshalNoEscapeBytes, err := marshalJSONNoHTMLEscape(inputMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(marshalNoEscapeBytes).To(Equal(expected))

			marshalBytes, err := json.Marshal(inputMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(marshalBytes).To(Equal(expected))

			Expect(marshalNoEscapeBytes).To(Equal(marshalBytes))
		})
	})
})
