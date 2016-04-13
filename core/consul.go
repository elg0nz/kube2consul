package api

import (
	"github.com/golang/glog"
	consulapi "github.com/hashicorp/consul/api"
)

type ConsulBackend struct {
	client *consulapi.Client
}

func NewConsulClient(consulAPI string) *ConsulBackend {
	cb := new(ConsulBackend)

	config := consulapi.DefaultConfig()
	config.Address = consulAPI

	if consulClient, err := consulapi.NewClient(config); err == nil {
		cb.client = consulClient
	} else {
		glog.Fatalln(err)
	}

	return cb
}

func (cb *ConsulBackend) Client() *consulapi.Client {
	return cb.client
}

func (cb *ConsulBackend) PutKV(key, value string) {
	kv := cb.client.KV()
	p := &consulapi.KVPair{Key: key, Value: []byte(value)}
	_, err := kv.Put(p, nil)
	if err != nil {
		glog.Fatalln("Cannot add value in Consul:", err)
	}
}

func (cb *ConsulBackend) GetKV(key string) (*consulapi.KVPair, error) {
	kv := cb.client.KV()
	value, _, err := kv.Get(key, nil)
	return value, err
}

func (cb *ConsulBackend) DeleteKV(key string) {
	kv := cb.client.KV()
	_, err := kv.Delete(key, nil)
	if err != nil {
		glog.Fatalln("Cannot add value in Consul:", err)
	}
}

func (cb *ConsulBackend) ListKV(key string) consulapi.KVPairs {
	kv := cb.client.KV()
	if values, _, err := kv.List(key, nil); err == nil {
		return values
	}
	return nil
}

// TODO: Utiliser les CatalogRegistration à la place ?
// https://godoc.org/github.com/hashicorp/consul/api#CatalogRegistration
func (cb *ConsulBackend) AddService(id, name, address string, port int, tags []string) {
	agent := cb.client.Agent()

	service := &consulapi.AgentServiceRegistration{
		ID:      id,
		Name:    name,
		Address: address,
		Port:    port,
		Tags:    tags,
	}

	if err := agent.ServiceRegister(service); err != nil {
		glog.Fatalln("Cannot register service:", err)
	}
}

func (cb *ConsulBackend) RemoveService(serviceID string) {
	agent := cb.client.Agent()

	if err := agent.ServiceDeregister(serviceID); err != nil {
		glog.Fatalln("Cannot deregister service:", err)
	}
}

func (cb *ConsulBackend) ListServices() map[string]*consulapi.AgentService {
	agent := cb.client.Agent()

	if services, err := agent.Services(); err == nil {
		return services
	} else {
		glog.Fatalln("Cannot deregister service:", err)
	}
	return nil
}
