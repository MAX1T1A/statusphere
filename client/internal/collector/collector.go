package collector

import (
	"statusphere-client/internal/models"
)

type Provider func(snap models.Snapshot)

type Collector struct {
	providers []Provider
}

func New(providers ...Provider) *Collector {
	return &Collector{providers: providers}
}

func (c *Collector) Collect() models.Snapshot {
	snap := make(models.Snapshot)
	for _, p := range c.providers {
		p(snap)
	}
	return snap
}
