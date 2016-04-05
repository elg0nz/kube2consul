package api

import (
	"github.com/golang/glog"

	kapi "k8s.io/kubernetes/pkg/api"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

type KubeWatcher struct {
	subscribers []Subscriber

	kubeClient *kclient.Client
}

type Subscriber struct {
	ch chan watch.Event
}

func NewKubeWatcher(cl *kclient.Client) *KubeWatcher {
	return &KubeWatcher{
		kubeClient: cl,
	}
}

func (kw *KubeWatcher) Start() {
	glog.Info("Start watching events")

	wsvc, _ := kw.kubeClient.Services(kapi.NamespaceAll).Watch(kapi.ListOptions{})
	wep, _ := kw.kubeClient.Endpoints(kapi.NamespaceAll).Watch(kapi.ListOptions{})

	events := make(chan watch.Event)

	go func() {
		for {
			select {
			case ev := <-wsvc.ResultChan():
				events <- ev
			case ev := <-wep.ResultChan():
				events <- ev
			}
		}
	}()

	// TODO: fix bug boucle infini !
	for event := range events {
		for _, subscriber := range kw.subscribers {
			subscriber.ch <- event
		}
	}
}

func (kw *KubeWatcher) Subscribe(ch chan watch.Event) {
	kw.subscribers = append(kw.subscribers, Subscriber{ch: ch})
}
