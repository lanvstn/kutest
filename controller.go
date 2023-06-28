package kutest

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/onsi/ginkgo/v2"
	"github.com/samber/lo"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/pointer"
)

func labels(name string, opts JobOptions) map[string]string {
	var labels map[string]string
	if opts.Labels != nil {
		labels = opts.Labels
	} else {
		labels = make(map[string]string)
	}

	labels["kutest.lanvstn.be/sessID"] = sessID
	labels["kutest.lanvstn.be/name"] = name

	return labels
}

func createJob(name string, opts JobOptions) error {
	uid := Config.UID
	labels := labels(name, opts)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Parallelism:             pointer.Int32(1),
			Completions:             pointer.Int32(1),
			BackoffLimit:            pointer.Int32(1),
			TTLSecondsAfterFinished: pointer.Int32(300),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
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
							Args:            ginkgoFocusFlags(),
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
								{
									Name: "KUTEST_PODNAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
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
			},
		},
	}

	if opts.Expose != nil {
		job.Spec.Template.Spec.Containers[0].Ports = []v1.ContainerPort{
			{
				Name:          "kutest",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: opts.Expose.Port,
			},
		}
	}

	if opts.ServiceAccount != "" {
		job.Spec.Template.Spec.ServiceAccountName = opts.ServiceAccount
	}

	if opts.MutateJob != nil {
		job = opts.MutateJob(job)
	}

	_, err := clientset.BatchV1().Jobs(opts.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func createService(name string, opts JobOptions) error {
	labels := labels(name, opts)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []v1.ServicePort{
				{
					Name:     "kutest",
					Protocol: corev1.ProtocolTCP,
					Port:     opts.Expose.Port,
				},
			},
		},
	}

	_, err := clientset.CoreV1().Services(opts.Namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	opts.Expose.Name <- name

	return nil
}

func deleteService(name string, opts JobOptions) error {
	err := clientset.CoreV1().Services(opts.Namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	return nil
}

func waitExit(jobName, namespace string) error {
	w, err := clientset.BatchV1().Jobs(namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", jobName),
	})
	if err != nil {
		return errors.Errorf("watch failure: %w", err)
	}

	for event := range w.ResultChan() {
		if event.Type == watch.Modified {
			job := event.Object.(*batchv1.Job)
			if job.Status.Failed > 0 && job.Status.Active == 0 && job.Status.Succeeded == 0 {
				return errors.New("job failed")
			} else if job.Status.Succeeded > 0 {
				return nil
			}
		}
	}

	return nil
}

func retrieveLogs(jobName, namespace string) ([]byte, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", batchv1.JobNameLabel, jobName),
	})
	if err != nil {
		return nil, errors.Errorf("list pods failure: %w", err)
	}

	podPhasePredicate := func(podPhase v1.PodPhase) func(pod v1.Pod, _ int) bool {
		return func(pod v1.Pod, _ int) bool {
			return pod.Status.Phase == podPhase
		}
	}

	candidatePods := append(lo.Filter[v1.Pod](pods.Items, podPhasePredicate(v1.PodSucceeded)),
		lo.Filter[v1.Pod](pods.Items, podPhasePredicate(v1.PodFailed))...)

	if len(candidatePods) == 0 {
		return nil, errors.New("tried to retrieve logs but no matching pods found in a final phase")
	}

	r, err := clientset.CoreV1().Pods(namespace).GetLogs(candidatePods[0].Name, &v1.PodLogOptions{}).Stream(context.Background())
	if err != nil {
		return nil, errors.Errorf("stream logs failure: %w", err)
	}

	return io.ReadAll(r)
}

// ginkgoFocusFlags generates Ginkgo focus flags to only focus on the current test
func ginkgoFocusFlags() []string {
	report := ginkgo.CurrentSpecReport()

	// filename is the base only, because the filepath as we see it from the container image may be different.
	// We can work around this by taking special care when building the container but I want to
	// keep the amount of "special things you have to know" as low as possible.
	filename := filepath.Base(report.LeafNodeLocation.FileName)

	return []string{
		"--ginkgo.focus-file", fmt.Sprintf("%v:%v", filename, report.LeafNodeLocation.LineNumber),

		// Focus is also given because we cannot be 100% sure that the filename+line is unique in the entire test suite
		"--ginkgo.focus", fmt.Sprintf("%v %v", strings.Join(report.ContainerHierarchyTexts, " "), report.LeafNodeText),
	}
}
