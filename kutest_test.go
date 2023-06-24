package kutest_test

import (
	"fmt"
	"testing"

	. "github.com/lanvstn/kutest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKutest(t *testing.T) {
	RegisterFailHandler(Fail)

	err := KutestSetup()
	if err != nil {
		t.Fatalf("kutest setup: %v", err)
	}

	RunSpecs(t, "Kutest Suite")
}

var _ = Describe("my tests", func() {
	Specify("something", func() {
		WithPod(PodOptions{
			Namespace: "default",
			Labels:    map[string]string{"foo": "bar"},
		}, func() {
			fmt.Println("hello from pod!")
		})

		fmt.Println("we're done!")
	})
})
