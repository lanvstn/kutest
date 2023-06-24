package kutest

import (
	"context"
	"fmt"
	"io"

	"github.com/go-errors/errors"
	"github.com/samber/lo"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/pointer"
)

func createJob(name string, opts JobOptions) error {
	uid := Config.UID

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: opts.Namespace,
			Labels:    opts.Labels,
		},
		Spec: batchv1.JobSpec{
			Parallelism:             pointer.Int32(1),
			Completions:             pointer.Int32(1),
			BackoffLimit:            pointer.Int32(1),
			TTLSecondsAfterFinished: pointer.Int32(300),
			Template: v1.PodTemplateSpec{
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
			},
		},
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
