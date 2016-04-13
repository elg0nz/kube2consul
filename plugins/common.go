package plugins

import (
	"github.com/golang/glog"

	"github.com/lightcode/kube2consul/core"
)

var plugins map[string]PluginEntry = make(map[string]PluginEntry)

type PluginEntry struct {
	name          string
	isInitialized bool
	plugin        Plugin
}

type Plugin interface {
	Initialize(*PluginManager)
	Sync()
}

func Register(name string, plugin Plugin) {
	glog.Infof("Register plugin \"%s\"", name)
	plugins[name] = PluginEntry{
		name:          name,
		plugin:        plugin,
		isInitialized: false,
	}
}

type PluginManager struct {
	Db          *api.Database
	Consul      *api.ConsulBackend
	KubeWatcher *api.KubeWatcher
}

func NewPluginManager(db *api.Database, cb *api.ConsulBackend, kw *api.KubeWatcher) *PluginManager {
	return &PluginManager{Db: db, Consul: cb, KubeWatcher: kw}
}

func (pm *PluginManager) Sync() {
	for _, e := range plugins {
		e.plugin.Sync()
	}
}

func (pm *PluginManager) Initialize() {
	for _, e := range plugins {
		if !e.isInitialized {
			e.plugin.Initialize(pm)
			e.isInitialized = true
		}
	}
}
