package kutest

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/ginkgo/v2"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	// Config is populated after KutestSetup
	Config ConfigSpec

	clientset *kubernetes.Clientset

	// sessID is the current session ID.
	// A new one is created by default unless specified in KUTEST_SESSID.
	// This happens on pods scheduled by Kutest.
	// controller determines whether this instance is the one that generated the sessID.
	sessID     string
	controller bool
)

type ConfigSpec struct {
	KubeconfigPath         string
	SessID                 string
	Image                  string `required:"true"`
	UID                    int64  `default:"1000"`
	DefaultImagePullPolicy string `default:"Always"`
}

type JobOptions struct {
	Namespace string
	Labels    map[string]string

	// MutateJob apply transformations to the pod that would be created
	MutateJob func(*batchv1.Job) *batchv1.Job
}

// WithJob runs f inside a new pod specified by PodOptions.
// If the pod fails or anything goes wrong it will call Fail on Ginkgo.
func WithJob(opts JobOptions, f func()) {
	ginkgo.GinkgoHelper()

	jobName := fmt.Sprintf("%s-%s", sessID, getShortTestID())

	hostname, err := os.Hostname()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("cannot get hostname: %v", err))
	}

	if strings.HasPrefix(hostname, jobName+"-") {
		// I am on the pod!
		f()
		return
	} else if !controller {
		return // not on the controller, nothing to see here
	}

	// Make the job, we are the controller.
	err = createJob(jobName, opts)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("pod creation failed: %v", err))
	}

	err = waitExit(jobName, opts.Namespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("pod failed: %v", err))
	}

	logs, err := retrieveLogs(jobName, opts.Namespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("cannot get logs: %v", err))
	}

	ginkgo.AddReportEntry("kutest-log-b64-"+jobName, base64.StdEncoding.EncodeToString(logs), ginkgo.ReportEntryVisibilityNever)
}

// getShortTestID returns an 8-char hash based on the current spec text
func getShortTestID() string {
	h := sha1.New()
	_, _ = h.Write([]byte(ginkgo.CurrentSpecReport().FullText()))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)[:9]
}
