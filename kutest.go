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
	KubeconfigPath         string
	SessID                 string
	Image                  string `required:"true"`
	UID                    int64  `default:"1000"`
	DefaultImagePullPolicy string `default:"Always"`
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
