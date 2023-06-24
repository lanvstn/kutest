package kutest

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func KutestSetup() error {
	err := envconfig.Process("kutest", &Config)
	if err != nil {
		return errors.New(err)
	}

	kubeconfig := Config.KubeconfigPath
	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); err != nil {
			return errors.Errorf("cannot stat kubeconfig at %q: %w", kubeconfig, err)
		}
	} else {
		// default to home if possible
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
			if _, err := os.Stat(kubeconfig); err != nil {
				kubeconfig = ""
			}
		}
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return errors.New(err)
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return errors.New(err)
	}

	if Config.SessID != "" {
		sessID = Config.SessID
	} else {
		sessID, controller = fmt.Sprintf("%x", rand.Int()), true
	}

	fmt.Printf("SessID: %v (controller: %v)\n", sessID, controller)

	return nil
}
