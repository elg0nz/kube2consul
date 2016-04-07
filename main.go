package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"

	"github.com/lightcode/kube2consul/api"
	"github.com/lightcode/kube2consul/plugins"

	// Plugins need to be imported for their init() to get executed and them to register
	_ "github.com/lightcode/kube2consul/plugins/services"
)

const ServiceLeaderKey = "lock/services_leader"

var (
	consulClient *api.ConsulBackend
	consulLock   *consulapi.Lock
	opts         CmdLineOpts
	sigch        chan os.Signal
)

type CmdLineOpts struct {
	kubeAPI   string
	consulAPI string
}

func init() {
	flag.StringVar(&opts.kubeAPI, "kubernetes-api", "http://127.0.0.1:8080", "Kubernetes API URL")
	flag.StringVar(&opts.consulAPI, "consul-api", "127.0.0.1:8500", "Consul API URL")
}

func run() {
	kubeWatcher := api.NewKubeWatcher(opts.kubeAPI)
	db := api.NewDatabase(opts.kubeAPI)
	pm := plugins.NewPluginManager(db, consulClient, kubeWatcher)

	pm.Initialize()
	db.UpdateDatabase()
	pm.Sync()

	ch := make(chan struct{})
	go db.StartWatching(ch)

	go kubeWatcher.Start()

	for {
		select {
		case s := <-sigch:
			if s == syscall.SIGHUP {
				glog.Info("User trigger an update")
				pm.Sync()
			}
		case <-ch:
			pm.Sync()
		}
	}
}

func attemptGetLock() <-chan struct{} {
	glog.Info("Attempting to get lock...")
	lockch, err := consulLock.Lock(nil)
	if err != nil {
		glog.Fatal(err)
	}

	glog.Info("This instance has got lock")

	return lockch
}

func releaseLock() {
	consulLock.Unlock()
	glog.Info("Lock has been released")
}

func main() {
	flag.Parse()

	consulClient = api.NewConsulClient(opts.consulAPI)

	consul := consulClient.Client()

	var err error
	consulLock, err = consul.LockOpts(&consulapi.LockOptions{
		Key:         ServiceLeaderKey,
		SessionName: "kube2consul lock",
	})
	if err != nil {
		glog.Fatal(err)
	}

	defer releaseLock()

	sigch = make(chan os.Signal, 1)
	signal.Notify(sigch,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

LOCK:
	lockch := attemptGetLock()

	go run()

	select {
	case <-lockch:
		goto LOCK
	case s := <-sigch:
		if s == syscall.SIGINT || s == syscall.SIGTERM || s == syscall.SIGQUIT {
			return
		}
	}
}
