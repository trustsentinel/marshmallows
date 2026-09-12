// Package registry is the broker's in-memory device catalogue: which devices are
// currently attached to this mini-cloud, keyed by their human-friendly id
// (cloud_id.device_id). It also renders the force-graph JSON the dashboard reads.
package registry

import (
	"sync"
	"time"
)

type Device struct {
	ID       string    `json:"id"` // cloud_id.device_id
	Name     string    `json:"name"`
	OS       string    `json:"os"`
	Addr     string    `json:"addr"`
	PubKey   string    `json:"pubkey,omitempty"`
	LastSeen time.Time `json:"last_seen"`
}

type Registry struct {
	mu      sync.Mutex
	cloud   string
	devices map[string]*Device
}

func New(cloud string) *Registry {
	return &Registry{cloud: cloud, devices: map[string]*Device{}}
}

func (r *Registry) Cloud() string { return r.cloud }

func (r *Registry) Add(d *Device) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[d.ID] = d
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.devices, id)
}

func (r *Registry) Get(id string) (*Device, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	return d, ok
}

func (r *Registry) List() []*Device {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Device, 0, len(r.devices))
	for _, d := range r.devices {
		out = append(out, d)
	}
	return out
}

// Graph renders the {nodes,links} force-graph the dashboard's /devices.json
// consumes: one node for the cloud (group 1) plus one per device (group 2),
// each linked to the cloud.
func (r *Registry) Graph() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	nodes := []map[string]any{{"id": r.cloud, "group": 1}}
	links := []map[string]any{}
	for id := range r.devices {
		nodes = append(nodes, map[string]any{"id": id, "group": 2})
		links = append(links, map[string]any{"source": r.cloud, "target": id, "value": 1})
	}
	return map[string]any{"nodes": nodes, "links": links}
}
