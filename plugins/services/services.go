package service

import (
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

func (sp *ServicePlugin) Sync() {
	exportedServices := make(ServiceList)

	services := sp.pm.Db.ListServices()

	for _, svc := range services.Items {
		ep := sp.pm.Db.GetEndpoints(svc.Name)
		exportedServices[svc.Name] = sp.createService(svc, *ep)
	}

	sp.updateKV(exportedServices)
	sp.updateDNS(exportedServices)
}

func (sp *ServicePlugin) handleEvent(event watch.Event) {
	const (
		service = iota
		endpoint
	)

	var (
		event_type int
		name       string
	)

	switch event.Object.(type) {
	case *kapi.Service:
		name = event.Object.(*kapi.Service).Name
		event_type = service
	case *kapi.Endpoints:
		name = event.Object.(*kapi.Endpoints).Name
		event_type = endpoint
	default:
		return
	}

	if event_type == endpoint && (event.Type == watch.Added || event.Type == watch.Modified) {
		if svc, err := sp.getServiceKV(name); err != nil {
			glog.Errorf("Cannot get service %s", name)
			return
		} else {
			glog.Infof("Endpoint %s modified or added", name)

			// Update svc with new endpoints
			ep := event.Object.(*kapi.Endpoints)
			svc.Endpoints = sp.getEnpointsIps(*ep)
			sp.updateServiceKV(svc)
			sp.updateServiceDNS(svc)
		}
	} else if event_type == service && event.Type == watch.Added {
		// Add new service in KV. We don't add endpoints because we don't
		// have any
		glog.Infof("Service %s added", name)

		kubeService := event.Object.(*kapi.Service)
		svc := sp.createService(*kubeService, kapi.Endpoints{})
		sp.updateServiceKV(svc)

	} else if event_type == service && event.Type == watch.Modified {
		if svc, err := sp.getServiceKV(name); err != nil {
			glog.Errorf("Cannot get service %s", name)
			return
		} else {
			// Update service with old endpoint
			glog.Infof("Service %s modified", name)

			ep := svc.Endpoints

			kubeService := event.Object.(*kapi.Service)
			svc = sp.createService(*kubeService, kapi.Endpoints{})
			svc.Endpoints = ep

			sp.updateServiceKV(svc)
			sp.updateServiceDNS(svc)
		}
	} else if event_type == service && event.Type == watch.Deleted {
		glog.Infof("Service %s deleted", name)

		kubeService := event.Object.(*kapi.Service)
		sp.removeServiceKV(kubeService.Name)
		sp.removeServiceDNS(kubeService.Name)

	} else {
		return
	}
}

func (sp *ServicePlugin) getEnpointsIps(ep kapi.Endpoints) (ips []string) {
	ips = make([]string, 0)

	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			ips = append(ips, addr.IP)
		}
	}

	return ips
}

func (sp *ServicePlugin) createService(svc kapi.Service, ep kapi.Endpoints) Service {
	ports := make(map[string]int)

	ips := sp.getEnpointsIps(ep)

	for _, port := range svc.Spec.Ports {
		ports[port.Name] = port.Port
	}

	se := Service{
		Name:        svc.Name,
		Annotations: svc.Annotations,
		Endpoints:   ips,
		Ports:       ports,
	}

	return se
}
