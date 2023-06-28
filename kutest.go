package kutest

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"runtime"

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
	// The path to the kubeconfig to use
	KubeconfigPath string

	// The image name which will be used by the tests to schedule copies of themselves to jobs
	Image string `required:"true"`

	// The UID of the user in the scheduled jobs
	UID int64 `default:"1000"`

	// The image pull policy set in the scheduled jobs.
	// Use `IfNotPresent` when using `kind load` and local images.
	DefaultImagePullPolicy string `default:"Always"`

	// SessID is the sessionID passed from the controller.
	// You should not set this yourself.
	SessID string

	// PodName is the name of this pod using Kubernetes envvar projection.
	// You should not set this yourself.
	PodName string
}

// shortID creates a short ID that is deterministic in the code location inside the same session.
// Don't call it from different code location if you want to get the same result.
func shortID() string {
	h := sha1.New()

	callers := make([]uintptr, 10)
	_ = runtime.Callers(2, callers)
	frames := runtime.CallersFrames(callers)

	for {
		frame, more := frames.Next()

		if frame.Function == "" {
			panic("function information not available in this build! required by Kutest for ID determination.")
		}

		h.Write([]byte(fmt.Sprintf("%s:%v", frame.Function, frame.Line)))

		if !more {
			break
		}
	}

	sum := h.Sum(nil)

	// That `k` is not a typo! We need valid DNS names which means no numbers in front.
	return fmt.Sprintf("k%s-%s", sessID, hex.EncodeToString(sum)[:9])
}
