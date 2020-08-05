package main

import (
	"fmt"
	"out-of-cluster-client-configuration/internal/app/service/infra/kubernetes"
)

func main() {
	var (
		clientSet, err = kubernetes.NewClientSet()
	)

	if err != nil {
		fmt.Println("ERROR! Unable to create new clientSet: ", err)
	} else {
		fmt.Println(clientSet)
	}
}