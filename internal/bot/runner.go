package bot

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"strconv"
)

type Runner interface {
	Start(ctx context.Context, opts *StartOptions) (RunnerHandle, error)
}

type KubernetesRunner struct {
	client *kubernetes.Clientset
}

func NewKubernetesRunner() (*KubernetesRunner, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesRunner{client}, nil
}

func (r *KubernetesRunner) Start(ctx context.Context, opts *StartOptions) (RunnerHandle, error) {
	// TODO move image name to config
	pods := r.pods()
	pod := toPod(opts, "mc-botnet-bot")

	pod, err := pods.Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return &kubernetesRunnerHandle{pod.Name, pods}, nil
}

func (r *KubernetesRunner) Stop(ctx context.Context) error {
	return r.pods().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
}

type RunnerHandle interface {
	Stop(ctx context.Context) error
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

func toPod(opts *StartOptions, image string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bot-" + opts.BotID.String(),
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:            "bot",
				Image:           image,
				ImagePullPolicy: corev1.PullNever,
				Env: []corev1.EnvVar{
					{
						Name:  "BOT_ID",
						Value: opts.BotID.String(),
					},
					{
						Name:  "BOT_HOST",
						Value: opts.McHost,
					},
					{
						Name:  "BOT_PORT",
						Value: strconv.Itoa(opts.McPort),
					},
					{
						Name:  "BOT_USERNAME",
						Value: opts.McUsername,
					},
					{
						Name:  "BOT_AUTH",
						Value: opts.McAuth,
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

	if opts.McToken != "" {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "BOT_TOKEN",
			Value: opts.McToken,
		})
	}

	return pod
}
