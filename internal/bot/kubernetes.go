package bot

import (
	"context"
	"strconv"

	"github.com/knadh/koanf/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// TODO fix Kubernetes runner

type KubernetesRunner struct {
	conf *koanf.Koanf

	client *kubernetes.Clientset
}

func NewKubernetesRunner(conf *koanf.Koanf) (*KubernetesRunner, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesRunner{conf.Cut("runner.kubernetes"), client}, nil
}

func (r *KubernetesRunner) Start(ctx context.Context, opts *RunnerOptions) (RunnerHandle, error) {
	pods := r.pods()
	pod := toPod(opts, r.conf)

	pod, err := pods.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return &kubernetesRunnerHandle{pod.Name, pods}, nil
}

func (r *KubernetesRunner) Stop(ctx context.Context) error {
	return r.pods().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
}

type kubernetesRunnerHandle struct {
	name string
	pods typedv1.PodInterface
}

func (k *kubernetesRunnerHandle) Stop(ctx context.Context) error {
	return k.pods.Delete(ctx, k.name, metav1.DeleteOptions{})
}

func (r *KubernetesRunner) pods() typedv1.PodInterface {
	return r.client.CoreV1().Pods("bot")
}

func toPod(opts *RunnerOptions, conf *koanf.Koanf) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bot-" + opts.ID.String(),
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicy(conf.MustString("image.restart")),
			Containers: []corev1.Container{{
				Name:            "bot",
				Image:           conf.MustString("image.name"),
				ImagePullPolicy: corev1.PullPolicy(conf.MustString("image.pull_policy")),
				Env: []corev1.EnvVar{
					{
						Name:  "BOT_ID",
						Value: opts.ID.String(),
					},
					{
						Name:  "GRPC_HOST",
						Value: opts.GRPCHost,
					},
					{
						Name:  "GRPC_PORT",
						Value: strconv.Itoa(opts.GRPCPort),
					},
				},
			}},
		},
	}

	return pod
}
