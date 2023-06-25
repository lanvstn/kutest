package kutest_test

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

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
	Specify("hello world", func() {
		WithJob(JobOptions{Namespace: "default"}, func() {
			fmt.Fprintln(GinkgoWriter, "hello from pod!")
		})
	})

	Specify("connectivity between two pods in the same namespace", func() {
		const (
			namespace string = "default"
			port      int32  = 8080
		)

		wg := &sync.WaitGroup{}

		wg.Add(1)
		nameCh := make(chan string, 1)

		go func() {
			defer wg.Done()
			defer GinkgoRecover()
			WithJob(JobOptions{
				Namespace: namespace,
				Expose: &ExposeOptions{
					Name: nameCh,
					Port: port,
				},
			}, func() {
				fmt.Println("hello from pod one! waiting for pod two to connect.")

				l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", port))
				Expect(err).ToNot(HaveOccurred())

				conn, err := l.Accept()
				Expect(err).ToNot(HaveOccurred())

				fmt.Printf("connected %v <- %v\n", conn.LocalAddr(), conn.RemoteAddr())
				conn.Close()
			})
		}()

		name := <-nameCh

		WithJob(JobOptions{
			Namespace: namespace,
		}, func() {
			fmt.Println("hello from pod two! connecting to pod one.")

			Eventually(func() error {
				conn, err := net.Dial("tcp", fmt.Sprintf("%s.%s.svc:%v", name, namespace, port))
				if err != nil {
					return err
				}

				fmt.Printf("connected %v -> %v\n", conn.LocalAddr(), conn.RemoteAddr())
				conn.Close()
				return nil
			}).WithTimeout(2 * time.Minute).Should(Succeed())
		})

		wg.Wait()
	})
})
