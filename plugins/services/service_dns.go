package service

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
)

const allServices = ""

func inSlice(value string, slice []string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

func generateServiceID(serviceName, portName, ipAddress string) string {
	return fmt.Sprintf("svc~%s~%s~%s", serviceName, portName, ipAddress)
}

func parseServiceID(id string) (serviceName, portName, ipAddress string, err error) {
	s := strings.SplitN(id, "~", 4)

	if len(s) != 4 {
		err = fmt.Errorf("Cannot parse service ID '%s'", id)
	} else {
		serviceName, portName, ipAddress = s[1], s[2], s[3]
	}

	return
}

func (sp *ServicePlugin) updateServiceDNS(svc Service) (ids []string) {
	ids = make([]string, 0)

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
			ids = append(ids, id)
		}
	}

	sp.cleanDNS(ids, svc.Name)

	return ids
}

func (sp *ServicePlugin) updateDNS(services ServiceList) {
	ids := make([]string, 0)

	for _, svc := range services {
		ids = append(ids, sp.updateServiceDNS(svc)...)
	}

	sp.cleanDNS(ids, allServices)

	glog.Info("Consul services resynced")
}

// serviceName peut être égale à un nom de service ou à allServices
func (sp *ServicePlugin) cleanDNS(ids []string, serviceName string) {
	invalidEntries := make([]string, 0)

	for id, kp := range sp.pm.Consul.ListServices() {
		if !inSlice(SERVICES_TAG, kp.Tags) {
			// Le service n'est pas managé par kube2consul
			continue
		}

		if serviceName != allServices {
			if name, _, _, err := parseServiceID(id); err != nil {
				invalidEntries = append(invalidEntries, id)
				continue
			} else if name != serviceName {
				continue
			}
		}

		if !inSlice(id, ids) {
			invalidEntries = append(invalidEntries, id)
		}
	}

	for _, id := range invalidEntries {
		glog.Infof("Remove service '%s' in Consul", id)
		sp.pm.Consul.RemoveService(id)
	}
}

func (sp *ServicePlugin) removeServiceDNS(serviceName string) {
	sp.cleanDNS([]string{}, serviceName)
}
