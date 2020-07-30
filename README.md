# kubernets-POC
This project shows comands to create and observe the deployments, pods, jobs, cronjobs and example for out-of-cluster-client-configurations using kubernets/client-go.

## Deployment
### Create deployment
- `kubectl create deployment NAME --image=image [--dry-run] [options]`
- example: `kubectl create deployment hello-node --image=k8s.gcr.io/echoserver:1.4`
### View deployments list 
- `kubectl get deployments`

## Jobs
### Create job
- write job specification file with `.yaml` extension. Demo file is available at `example/job.yaml`
- `kubectl apply (-f FILENAME | -k DIRECTORY) [options]`
- example : `kubectl apply -f ./example/job.yaml`
### View jobs list
- `kubectl get jobs`
### View job details
- `kubectl describe jobs/<job-name>`

## Cronjobs
### Create cronjob
- write cronjob specification in file with `.yaml` extension. Demo file is available at `example/cronjob.yaml`
- `kubectl create -f FILENAME [options]`
- example: `kubectl create -f ./example/cronjob.yaml`
### View cronjobs list
- `kubectl get cronjobs`
### View progress of cronjob
- `kubectl get jobs --watch`

## Run client-go example out-of-cluster-client-configurations
- `git clone https://github.com/gopay-bootcamp/kubernets-POC.git`
- `cd kubernets-POC`
- `go build -o output .`
- `./output`

## Requirements
- Docker
- Kubectl
- Minikube
