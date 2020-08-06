package main

import (
	"fmt"
	"net/http"
	"out-of-cluster-client-configuration/internal/app/service/infra/kubernetes"
)

func main() {
	var (
		httpClient = http.Client{
			Timeout:       0,
		}
		newClient, err = kubernetes.NewKubernetesClient(&httpClient)
	)

	if err != nil {
		fmt.Println("ERROR! Unable to create new clientSet: ", err)
	} else {
		fmt.Println("New Kube client created: ", newClient)
	}

}