package kubernetes

import (
	"context"
	"errors"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"io"
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"path/filepath"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeRestClient "k8s.io/client-go/rest"
	"out-of-cluster-client-configuration/internal/app/service/infra/config"
)

var (
	typeMeta meta.TypeMeta
	namespace string
	timeoutError = errors.New("timeout when waiting job to be available")
)

func init() {
	typeMeta = meta.TypeMeta{
		Kind: "Job",
		APIVersion: "batch/v1",
	}
	namespace = "default"
}

type kubernetesClient struct {
	clientSet kubernetes.Interface
	httpClient *http.Client
}

type KubernetesClient interface {
	ExecuteJobWithCommand(imagename string, args map[string]string, commands []string) (string, error)
	ExecuteJob(imagename string, args map[string]string) (string, error)
	JobExecutionStatus(jobName string) (string, error)
	GetPodLogs(pod *v1.Pod) (io.ReadCloser, error)
}

func NewClientSet() (*kubernetes.Clientset, error) {
	var kubeConfig *kubeRestClient.Config
	fmt.Println(config.Config().KubeConfig)
	if config.Config().KubeConfig == "out-of-cluster" {
		//logger.Info("service is running outside kube cluster")
		fmt.Println("service is running outside kube cluster")
		home := os.Getenv("HOME")

		kubeConfigPath := filepath.Join(home, ".kube", "config")

		configOverrides := &clientcmd.ConfigOverrides{}
		if config.Config().KubeContext != "default" {
			configOverrides.CurrentContext = config.Config().KubeContext
		}

		var err error
		kubeConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
			configOverrides).ClientConfig()
		if err != nil {
			return nil, err
		}

	} else {
		var err error
		kubeConfig, err = kubeRestClient.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return clientSet, nil
}

func uniqueName() string {
	return "proctor" + "-" + uuid.NewV4().String()
}

func jobLabel(executionName string) map[string]string {
	return map[string]string{
		"job": executionName,
	}
}

func getEnvVars(envMap map[string]string) []v1.EnvVar {
	var envVars []v1.EnvVar
	for k, v := range envMap {
		envVar := v1.EnvVar{
			Name:  k,
			Value: v,
		}
		envVars = append(envVars, envVar)
	}
	return envVars
}

func (client *kubernetesClient) ExecuteJobWithCommand(imageName string, envMap map[string]string, command []string) (string, error) {
	executionName := uniqueName()
	label := jobLabel(executionName)

	batchV1 := client.clientSet.BatchV1()
	kubernetesJobs := batchV1.Jobs(namespace)

	container := v1.Container{
		Name:  executionName,
		Image: imageName,
		Env:   getEnvVars(envMap),
	}

	if len(command) != 0 {
		container.Command = command
	}

	podSpec := v1.PodSpec{
		Containers:         []v1.Container{container},
		RestartPolicy:      v1.RestartPolicyNever,
		ServiceAccountName: config.Config().KubeServiceAccountName,
	}

	objectMeta := meta.ObjectMeta{
		Name:        executionName,
		Labels:      label,
		Annotations: config.Config().JobPodAnnotations,
	}

	template := v1.PodTemplateSpec{
		ObjectMeta: objectMeta,
		Spec:       podSpec,
	}

	jobSpec := batch.JobSpec{
		Template:              template,
		ActiveDeadlineSeconds: config.Config().KubeJobActiveDeadlineSeconds,
		BackoffLimit:          config.Config().KubeJobRetries,
	}

	jobToRun := batch.Job{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec:       jobSpec,
	}

	var (
		ctx context.Context
		cancel context.CancelFunc
	)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	_, err := kubernetesJobs.Create(ctx, &jobToRun, meta.CreateOptions{})
	if err != nil {
		return "", err
	}
	return executionName, nil
}