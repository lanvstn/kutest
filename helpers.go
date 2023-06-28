package kutest

import (
	"encoding/base64"
	"fmt"
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
// Please be aware that since f executes on another pod, you are not able to pass values outside of f!
//
// - When on a controller, this runs the job to completion.
//
// - When on the job, this runs f.
//
// - When on another job, this does nothing.
func WithJob(opts JobOptions, f func()) {
	ginkgo.GinkgoHelper()

	id := shortID()
	selectedPod := strings.HasPrefix(Config.PodName, id+"-")

	if selectedPod {
		fmt.Fprintf(ginkgo.GinkgoWriter, "Running WithJob function as %q\n", id)
		f() // I am on the pod!
	}

	if !selectedPod && !controller {
		fmt.Fprintf(ginkgo.GinkgoWriter,
			"WithJob function execution skipped, we are %q but this WithJob wants to run on pods for job %q. "+
				"This is completely normal if there are multiple WithJob calls in one test.\n",
			Config.PodName, id)
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
	err := createJob(id, opts)
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
