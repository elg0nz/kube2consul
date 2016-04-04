package backend

import (
	"flag"

	"github.com/golang/glog"
	consul "github.com/hashicorp/consul/api"
)

var consulApiUrl string

func init() {
	flag.StringVar(&consulApiUrl, "consul-api", "127.0.0.1:8500", "Consul API URL")
}

type ConsulBackend struct {
	client *consul.Client
}

func NewConsulClient() *ConsulBackend {
	cb := new(ConsulBackend)

	consulConfig := &consul.Config{
		Address: consulApiUrl,
	}

	if consulClient, err := consul.NewClient(consulConfig); err == nil {
		cb.client = consulClient
	} else {
		glog.Fatalln(err)
	}

	return cb
}

func (cb *ConsulBackend) PutKV(key, value string) {
	kv := cb.client.KV()
	p := &consul.KVPair{Key: key, Value: []byte(value)}
	_, err := kv.Put(p, nil)
	if err != nil {
		glog.Fatalln("Cannot add value in Consul:", err)
	}
}

func (cb *ConsulBackend) GetKV(key string) (*consul.KVPair, error) {
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

func (cb *ConsulBackend) ListKV(key string) consul.KVPairs {
	kv := cb.client.KV()
	if values, _, err := kv.List(key, nil); err == nil {
		return values
	}
	return nil
}

func (cb *ConsulBackend) AddService(id, name, address string, port int, tags []string) {
	agent := cb.client.Agent()

	service := &consul.AgentServiceRegistration{
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

func (cb *ConsulBackend) ListServices() map[string]*consul.AgentService {
	agent := cb.client.Agent()

	if services, err := agent.Services(); err == nil {
		return services
	} else {
		glog.Fatalln("Cannot deregister service:", err)
	}
	return nil
}
