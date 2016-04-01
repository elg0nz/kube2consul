package plugins

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/kubernetes/pkg/watch"

	"github.com/golang/glog"
	"github.com/lightcode/kube2consul/plugins"

	kapi "k8s.io/kubernetes/pkg/api"
)

const (
	SERVICES_ROOT = "services"
	SERVICES_TAG  = "kube2consul-service-managed"
)

type ServiceList map[string]Service

type ServicePlugin struct {
	pm *plugins.PluginManager
}

type Service struct {
	Name        string            `json:"name"`
	Annotations map[string]string `json:"annotations"`
	Endpoints   []string          `json:"endpoints"`
	Ports       map[string]int    `json:"ports"`
}

func init() {
	s := new(ServicePlugin)
	plugins.Register("services", s)
}

func (sp *ServicePlugin) Initialize(pm *plugins.PluginManager) {
	sp.pm = pm

	ch := make(chan watch.Event)
	pm.Db.Subscribe(ch)

	go func() {
		for event := range ch {
			switch event.Object.(type) {
			case *kapi.Service, *kapi.Endpoints:
				sp.handleEvent(event)
			}
		}
	}()
}

func (sp *ServicePlugin) handleEvent(event watch.Event) {
	glog.Infof("New event: %v", event)
}

func (sp *ServicePlugin) Sync() {
	exportedServices := make(ServiceList)

	services := sp.pm.Db.ListServices()

	for _, svc := range services.Items {
		ips := make([]string, 0)
		ports := make(map[string]int)

		ep := sp.pm.Db.GetEndpoints(svc.Name)

		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				ips = append(ips, addr.IP)
			}
		}

		for _, port := range svc.Spec.Ports {
			ports[port.Name] = port.Port
		}

		se := Service{
			Name:        svc.Name,
			Annotations: svc.Annotations,
			Endpoints:   ips,
			Ports:       ports,
		}

		exportedServices[se.Name] = se
	}

	sp.updateKV(exportedServices)
	sp.updateDNS(exportedServices)
}

func (sp *ServicePlugin) updateKV(services ServiceList) {
	for _, kp := range sp.pm.Consul.ListKV(SERVICES_ROOT) {
		s := strings.Split(kp.Key, "/")
		serviceName := s[len(s)-1]
		if _, ok := services[serviceName]; !ok {
			fmt.Println("delete", kp.Key)
			sp.pm.Consul.DeleteKV(kp.Key)
		}
	}

	for _, svc := range services {
		obj, _ := json.Marshal(svc)
		sp.pm.Consul.PutKV(fmt.Sprintf("%s/%s", SERVICES_ROOT, svc.Name), string(obj))
	}

	glog.Info("Consul KV resynced")
}

func generateServiceID(serviceName, portName, ipAddress string) string {
	return fmt.Sprintf("svc-%s-%s-%s", serviceName, portName, ipAddress)
}

func (sp *ServicePlugin) updateDNS(services ServiceList) {
	servicesListID := make(map[string]bool)

	for _, svc := range services {
		for _, ipAddress := range svc.Endpoints {
			for portName, portNumber := range svc.Ports {
				id := generateServiceID(svc.Name, portName, ipAddress)
				sp.pm.Consul.AddService(
					id,
					fmt.Sprintf("%s-%s", svc.Name, portName),
					ipAddress,
					portNumber,
					[]string{SERVICES_TAG},
				)
				servicesListID[id] = true
			}
		}
	}

	for k, v := range sp.pm.Consul.ListServices() {
		if inSlice(SERVICES_TAG, v.Tags) {
			if _, ok := servicesListID[k]; !ok {
				sp.pm.Consul.RemoveService(k)
				glog.Infof("Remove service '%s' in Consul", k)
			}
		}
	}

	glog.Info("Consul services resynced")
}

func inSlice(value string, slice []string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}
