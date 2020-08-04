package main

import (
	"fmt"
	"os"
	kclient "out-of-cluster-client-configuration/internal/app/service/infra/kubernetes/http"
)

func main() {
	clientSet, err := kclient.NewClientSet()
	if err != nil {
		fmt.Println("ERROR! Unable to create new clientSet: ", err)
	} else {
		fmt.Println(clientSet)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
