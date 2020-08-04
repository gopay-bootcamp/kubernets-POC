package kubernetes

import (
	"errors"
	"honnef.co/go/tools/config"
	"io"
	"net/http"

	"k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeRestClient "k8s.io/client-go/rest"
)

var (
	typeMeta meta.TypeMeta
	namespace string
	timeoutError = errors.New("timeout when waiting job to be available")
	config2  map[string]string
)

func init() {
	typeMeta = meta.TypeMeta{
		Kind: "Job",
		APIVersion: "batch/v1",
	}
	namespace = "default"
	config2 = map[string]string{
		"namespace" : "default",
	}
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
	kubeConfig, err := kubeRestClient.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientSet, err := kubernetes.NewForConfig(kubeConfig)

	return clientSet, nil
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

	_, err := kubernetesJobs.Create(&jobToRun)
	if err != nil {
		return "", err
	}
	return executionName, nil
}