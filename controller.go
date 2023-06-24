package kutest

import (
	"context"
	"fmt"

	"github.com/go-errors/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/pointer"
)

func runPod(podName string, opts PodOptions) error {
	uid := Config.UID

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: opts.Namespace,
			Labels:    opts.Labels,
		},
		Spec: v1.PodSpec{
			SecurityContext: &v1.PodSecurityContext{
				RunAsNonRoot: pointer.Bool(true),
				RunAsUser:    pointer.Int64(uid),
				RunAsGroup:   pointer.Int64(uid),
				FSGroup:      pointer.Int64(uid),
			},
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:            "kutest",
					Image:           Config.Image,
					ImagePullPolicy: v1.PullPolicy(Config.DefaultImagePullPolicy),
					Env: []v1.EnvVar{
						{
							Name:  "KUTEST_IMAGE",
							Value: Config.Image,
						},
						{
							Name:  "KUTEST_SESSID",
							Value: sessID,
						},
					},
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("10m"),
							v1.ResourceMemory: resource.MustParse("40M"),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("10m"),
							v1.ResourceMemory: resource.MustParse("40M"),
						},
					},
					SecurityContext: &v1.SecurityContext{
						Privileged:               pointer.Bool(false),
						AllowPrivilegeEscalation: pointer.Bool(false),
						RunAsNonRoot:             pointer.Bool(true),
						RunAsUser:                pointer.Int64(uid),
						RunAsGroup:               pointer.Int64(uid),
						ReadOnlyRootFilesystem:   pointer.Bool(true),
						Capabilities: &v1.Capabilities{
							Drop: []v1.Capability{
								"ALL",
							},
						},
					},
				},
			},
		},
	}

	if opts.MutatePod != nil {
		pod = opts.MutatePod(pod)
	}

	_, err := clientset.CoreV1().Pods(opts.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func waitExit(podName, namespace string) error {
	w, err := clientset.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", podName),
	})
	if err != nil {
		return fmt.Errorf("watch failure: %w", err)
	}

	for event := range w.ResultChan() {
		if event.Type == watch.Modified {
			switch event.Object.(*v1.Pod).Status.Phase {
			case v1.PodFailed:
				return errors.New("pod failed")
			case v1.PodSucceeded:
				return nil
			}
		}
	}

	return nil
}
