package cache

import (
	"sync"
	"time"

	"go-monitoring/models"
)

// MetricsCache gÃ¨re le cache des mÃ©triques des machines
type MetricsCache struct {
	machines map[string]cachedMachine
	mu       sync.RWMutex
	ttl      time.Duration
}

type cachedMachine struct {
	Machine    models.Machine
	Expiration time.Time
}

// NewMetricsCache crÃ©e un nouveau cache
func NewMetricsCache(ttl time.Duration) *MetricsCache {
	c := &MetricsCache{
		machines: make(map[string]cachedMachine),
		ttl:      ttl,
	}

	// DÃ©marrer le nettoyage en arriÃ¨re-plan
	go c.cleanup()

	return c
}

// Set met Ã  jour les infos d'une machine
func (c *MetricsCache) Set(machine models.Machine) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.machines[machine.ID] = cachedMachine{
		Machine:    machine,
		Expiration: time.Now().Add(c.ttl),
	}
}

// Get rÃ©cupÃ¨re les infos d'une machine
func (c *MetricsCache) Get(id string) (models.Machine, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.machines[id]
	if !found {
		return models.Machine{}, false
	}

	if time.Now().After(item.Expiration) {
		return models.Machine{}, false
	}

	return item.Machine, true
}

// GetLastKnown rÃ©cupÃ¨re la derniÃ¨re valeur connue (mÃªme expirÃ©e)
func (c *MetricsCache) GetLastKnown(id string) (models.Machine, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.machines[id]
	return item.Machine, found
}

// Invalidate invalide une entrÃ©e spÃ©cifique
func (c *MetricsCache) Invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.machines, id)
}

// cleanup nettoie les Ã©lÃ©ments expirÃ©s
func (c *MetricsCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for id, item := range c.machines {
			if now.After(item.Expiration) {
				delete(c.machines, id)
			}
		}
		c.mu.Unlock()
	}
}
