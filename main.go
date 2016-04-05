package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"

	"github.com/lightcode/kube2consul/api"
	"github.com/lightcode/kube2consul/plugins"

	// Plugins need to be imported for their init() to get executed and them to register
	_ "github.com/lightcode/kube2consul/plugins/services"
)

var (
	kubeAPIServerURL string
)

func init() {
	flag.StringVar(&kubeAPIServerURL, "kubernetes-api", "", "Kubernetes API URL")
}

func getKubeClient() *kclient.Client {
	config := &restclient.Config{
		Host: kubeAPIServerURL,
	}

	if kubeClient, err := kclient.New(config); err == nil {
		return kubeClient
	} else {
		glog.Fatalln("Can't connect to Kubernetes API:", err)
	}
	return nil
}

func handleArgs() {
	flag.Parse()

	if kubeAPIServerURL == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	handleArgs()

	consulClient := api.NewConsulClient()
	kubeWatcher := api.NewKubeWatcher(getKubeClient())

	db := api.NewDatabase(getKubeClient())
	pm := plugins.NewPluginManager(db, consulClient, kubeWatcher)

	pm.Initialize()
	db.UpdateDatabase()
	pm.Sync()

	ch := make(chan struct{})
	go db.StartWatching(ch)

	go kubeWatcher.Start()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP)

	for {
		select {
		case <-sigc:
			glog.Info("User trigger an update")
			pm.Sync()
		case <-ch:
			pm.Sync()
		}
	}
}
