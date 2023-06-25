package kutest

import (
	"crypto/sha1"
	"encoding/hex"

	"github.com/onsi/ginkgo/v2"
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

// getShortTestID returns an 8-char hash based on the current spec text
func getShortTestID() string {
	h := sha1.New()
	_, _ = h.Write([]byte(ginkgo.CurrentSpecReport().FullText()))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)[:9]
}
