package handlers

import (
	"fmt"
	"testing"
	"time"

	"go-monitoring/cache"
	"go-monitoring/config"
	"go-monitoring/models"
)

func BenchmarkCollectAllMachines_CacheHit(b *testing.B) {
	// Setup with many machines
	count := 100
	machines := make([]config.MachineConfig, count)
	c := cache.NewMetricsCache(1 * time.Minute)

	for i := 0; i < count; i++ {
		id := fmt.Sprintf("machine-%d", i)
		machines[i] = config.MachineConfig{ID: id, Host: "1.2.3.4"}
		c.Set(models.Machine{ID: id, Status: "online"})
	}

	cfg := &config.Config{Machines: machines}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectAllMachines(cfg, nil, c, 1*time.Second, false)
	}
}
