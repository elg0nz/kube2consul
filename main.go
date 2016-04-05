package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"

	"github.com/lightcode/kube2consul/api"
	"github.com/lightcode/kube2consul/plugins"

	// Plugins need to be imported for their init() to get executed and them to register
	_ "github.com/lightcode/kube2consul/plugins/services"
)

var opts CmdLineOpts

type CmdLineOpts struct {
	kubeAPI   string
	consulAPI string
}

func init() {
	flag.StringVar(&opts.kubeAPI, "kubernetes-api", "http://127.0.0.1:8080", "Kubernetes API URL")
	flag.StringVar(&opts.consulAPI, "consul-api", "127.0.0.1:8500", "Consul API URL")
}

func main() {
	flag.Parse()

	consulClient := api.NewConsulClient(opts.consulAPI)
	kubeWatcher := api.NewKubeWatcher(opts.kubeAPI)

	db := api.NewDatabase(opts.kubeAPI)
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
