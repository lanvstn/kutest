package kutest

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/ginkgo/v2"
	batchv1 "k8s.io/api/batch/v1"
)

type JobOptions struct {
	Namespace string
	Labels    map[string]string

	// MutateJob apply transformations to the pod that would be created
	MutateJob func(*batchv1.Job) *batchv1.Job

	// Expose, if set, will create a service for the job.
	Expose *ExposeOptions
}

type ExposeOptions struct {
	// Name is REQUIRED and will recieve the name of the service (when created if on controller)
	//
	// TODO:
	// 	This feels slightly awkward but is due the fact that the name creation is a Kutest internal detail.
	// 	Maybe this should be reconsidered to create a more natural interface.
	Name chan<- string

	// Port is both the service and the target port.
	Port int32
}

// WithJob runs f inside a new pod specified by PodOptions.
// If the pod fails or anything goes wrong it will call Fail on Ginkgo.
func WithJob(opts JobOptions, f func()) {
	ginkgo.GinkgoHelper()

	id := shortID()

	hostname, err := os.Hostname()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("cannot get hostname: %v", err))
	}

	if strings.HasPrefix(hostname, id+"-") {
		fmt.Fprintln(ginkgo.GinkgoWriter, "Running WithJob function")
		f() // I am on the pod!
	}

	if !controller {
		if opts.Expose != nil {
			// Send service name without making it
			opts.Expose.Name <- id
		}
		return
	}

	fmt.Fprintln(ginkgo.GinkgoWriter, "Creating job")

	// Make the job, we are the controller.
	err = createJob(id, opts)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("pod creation failed: %v", err))
	}

	if opts.Expose != nil {
		err := createService(id, opts)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("service creation failed: %v", err))
		}
	}

	fmt.Fprintln(ginkgo.GinkgoWriter, "Waiting for job exit")

	exitErr := waitExit(id, opts.Namespace)

	fmt.Fprintln(ginkgo.GinkgoWriter, "Cleaning up")

	if opts.Expose != nil {
		err := deleteService(id, opts)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("service deletion failed: %v", err))
		}
	}

	logs, err := retrieveLogs(id, opts.Namespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("cannot get logs: %v", err))
	}

	ginkgo.AddReportEntry("kutest-log-b64-"+id, base64.StdEncoding.EncodeToString(logs), ginkgo.ReportEntryVisibilityNever)

	fmt.Fprintln(ginkgo.GinkgoWriter, "WithJob controller all done.")

	if exitErr != nil {
		ginkgo.Fail(fmt.Sprintf("pod failed: %v", err))
	}
}
