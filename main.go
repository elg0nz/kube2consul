package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"

	"github.com/lightcode/kube2consul/backend"
	"github.com/lightcode/kube2consul/database"
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

func getKubeClient(config *restclient.Config) *kclient.Client {
	if kubeClient, err := kclient.New(config); err == nil {
		return kubeClient
	} else {
		glog.Fatalln("Can't connect to Kubernetes API:", err)
	}
	return nil
}

// USELESS ?
func getKubeExtClient(config *restclient.Config) *kclient.ExtensionsClient {
	if kubeExtClient, err := kclient.NewExtensions(config); err == nil {
		return kubeExtClient
	} else {
		glog.Fatalln("Can't connect to Kubernetes API:", err)
	}
	return nil
}

func main() {
	flag.Parse()

	if kubeAPIServerURL == "" {
		flag.Usage()
		os.Exit(1)
	}

	glog.Infof("Kubernetes API URL: %s", kubeAPIServerURL)

	config := &restclient.Config{
		Host: kubeAPIServerURL,
	}

	cb := backend.NewConsulClient()

	db := database.NewDatabase(getKubeClient(config))
	pm := plugins.NewPluginManager(db, cb)

	pm.Initialize()
	db.UpdateDatabase()
	pm.Sync()

	ch := make(chan struct{})
	go db.StartWatching(ch)

	go db.WatchEvents()

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
