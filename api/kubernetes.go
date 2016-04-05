package api

import (
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
)

func getKubeClient(kubeAPI string) *kclient.Client {
	config := &restclient.Config{
		Host: kubeAPI,
	}

	if kubeClient, err := kclient.New(config); err == nil {
		return kubeClient
	} else {
		glog.Fatalln("Can't connect to Kubernetes API:", err)
	}
	return nil
}
