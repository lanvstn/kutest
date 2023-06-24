package kutest

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
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

type PodOptions struct {
	Namespace string
	Labels    map[string]string

	// MutatePod apply transformations to the pod that would be created
	MutatePod func(*v1.Pod) *v1.Pod
}

// WithPod runs f inside a new pod specified by PodOptions.
// If the pod fails or anything goes wrong it will call Fail on Ginkgo.
func WithPod(opts PodOptions, f func()) {
	ginkgo.GinkgoHelper()

	podName := fmt.Sprintf("%s-%s", sessID, getShortTestID())

	hostname, err := os.Hostname()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("cannot get hostname: %v", err))
	}

	if hostname == podName {
		// I am on the pod!
		f()
		return
	} else if !controller {
		return // not on the controller, nothing to see here
	}

	// Make the pod, we are the controller.
	err = runPod(podName, opts)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("pod creation failed: %v", err))
	}

	err = waitExit(podName, opts.Namespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("pod failed: %v", err))
	}
}

// getShortTestID returns an 8-char hash based on the current spec text
func getShortTestID() string {
	h := sha1.New()
	_, _ = h.Write([]byte(ginkgo.CurrentSpecReport().FullText()))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)[:9]
}
