package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
)

func (sp *ServicePlugin) updateServiceKV(svc Service) {
	obj, _ := json.Marshal(svc)
	sp.pm.Consul.PutKV(fmt.Sprintf("%s/%s", SERVICES_ROOT, svc.Name), string(obj))
}

func (sp *ServicePlugin) updateKV(services ServiceList) {
	for _, kp := range sp.pm.Consul.ListKV(SERVICES_ROOT) {
		s := strings.Split(kp.Key, "/")
		serviceName := s[len(s)-1]
		if _, ok := services[serviceName]; !ok {
			sp.pm.Consul.DeleteKV(kp.Key)
		}
	}

	for _, svc := range services {
		sp.updateServiceKV(svc)
	}

	glog.Info("Consul KV resynced")
}

func (sp *ServicePlugin) getServiceKV(serviceName string) (svc Service, _ error) {
	key := fmt.Sprintf("%s/%s", SERVICES_ROOT, serviceName)

	if kp, err := sp.pm.Consul.GetKV(key); err != nil {
		return svc, err
	} else if err := json.Unmarshal(kp.Value, &svc); err != nil {
		return svc, err
	} else {
		return svc, nil
	}
}

func (sp *ServicePlugin) removeServiceKV(serviceName string) {
	key := fmt.Sprintf("%s/%s", SERVICES_ROOT, serviceName)
	sp.pm.Consul.DeleteKV(key)
}
