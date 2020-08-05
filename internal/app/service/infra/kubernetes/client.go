package kubernetes

import (
	"context"
	"errors"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"io"
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
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
	ExecuteJobWithCommand(imageName string, args map[string]string, commands []string) (string, error)
	ExecuteJob(imageName string, args map[string]string) (string, error)
	JobExecutionStatus(jobName string) (string, error)
	GetPodLogs(pod *v1.Pod) (io.ReadCloser, error)
}

func NewClientSet() (*kubernetes.Clientset, error) {
	var kubeConfig *kubeRestClient.Config

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

func jobLableSelector(executionName string) string {
	return fmt.Sprintf("job=%s", executionName)
}

func (client *kubernetesClient) JobExecutionStatus(executionName string) (string, error) {
	batchV1 := client.clientSet.BatchV1()
	kubernetsJobs := batchV1.Jobs(namespace)
	listOptions := meta.ListOptions {
		TypeMeta:	typeMeta,
		LabelSelector: jobLableSelector(executionName),
	}

	var (
		ctx context.Context
		cancel context.CancelFunc
	)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	watchJob, err := kubernetsJobs.Watch(ctx, listOptions)
	if err != nil {
		return "FAILED", err
	}

	resultChan := watchJob.ResultChan()
	defer watchJob.Stop()
	var event watch.Event
	var jobEvent *batch.Job

	for event = range resultChan {
		if event.Type == watch.Error {
			return "JOB_EXECUTION_STATUS_FETCH_ERROR", nil
		}

		jobEvent = event.Object.(*batch.Job)
		if jobEvent.Status.Succeeded >= int32(1) {
			return "SUCCEEDED", nil
		} else if jobEvent.Status.Failed >= int32(1) {
			return "FAILED", nil
		}
	}

	return "NO_DEFINITIVE_JOB_EXECUTION_STATUS_FOUND", nil
}

func (client *kubernetesClient) GetPodLogs(pod *v1.Pod) (io.ReadCloser, error) {
	//logger.Debug("reading pod logs for: ", pod.Name)
	podLogOpts := v1.PodLogOptions{
		Follow: true,
	}

	var (
		ctx context.Context
		cancel context.CancelFunc
	)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	request := client.clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	response, err := request.Stream(ctx)

	if err != nil {
		return nil, err
	}
	return response, nil
}
