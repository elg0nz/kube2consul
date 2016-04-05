package api

import (
	"sync"
	"time"

	"github.com/golang/glog"

	kapi "k8s.io/kubernetes/pkg/api"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
)

const updateInterval = time.Minute * 10

type Database struct {
	services  *kapi.ServiceList
	endpoints *kapi.EndpointsList

	kubeClient *kclient.Client

	sync.Mutex
}

func NewDatabase(cl *kclient.Client) *Database {
	return &Database{
		kubeClient: cl,
	}
}

func (db *Database) UpdateDatabase() {
	db.Lock()
	defer db.Unlock()

	if services, err := db.kubeClient.Services(kapi.NamespaceAll).List(kapi.ListOptions{}); err == nil {
		db.services = services
	} else {
		glog.Errorf("Cannot get service list: %s", err)
	}

	if endpoints, err := db.kubeClient.Endpoints(kapi.NamespaceAll).List(kapi.ListOptions{}); err == nil {
		db.endpoints = endpoints
	} else {
		glog.Errorf("Cannot get endpoints list: %s", err)
	}
}

func (db *Database) ListServices() (services *kapi.ServiceList) {
	db.Lock()
	services = db.services
	db.Unlock()
	return services
}

func (db *Database) ListEndpoints() (endpoints *kapi.EndpointsList) {
	db.Lock()
	endpoints = db.endpoints
	db.Unlock()
	return endpoints
}

func (db *Database) GetEndpoints(name string) *kapi.Endpoints {
	for _, ep := range db.ListEndpoints().Items {
		if ep.Name == name {
			return &ep
		}
	}
	return nil
}

func (db *Database) StartWatching(ch chan struct{}) {
	for range time.NewTicker(updateInterval).C {
		db.UpdateDatabase()
		ch <- struct{}{}
	}
}
