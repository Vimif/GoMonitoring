package cache

import (
	"sync"
	"testing"
	"time"

	"go-monitoring/models"
)

func TestNewMetricsCache(t *testing.T) {
	ttl := 1 * time.Minute
	c := NewMetricsCache(ttl)

	if c == nil {
		t.Fatal("NewMetricsCache returned nil")
	}

	if c.machines == nil {
		t.Error("MetricsCache.machines map was not initialized")
	}

	if c.ttl != ttl {
		t.Errorf("Expected TTL %v, got %v", ttl, c.ttl)
	}
}

func TestSetGet(t *testing.T) {
	c := NewMetricsCache(1 * time.Minute)

	machine := models.Machine{
		ID:   "machine-1",
		Name: "Test Machine",
		Host: "localhost",
	}

	c.Set(machine)

	// Test Get existing
	retrieved, found := c.Get("machine-1")
	if !found {
		t.Error("Expected to find machine-1")
	}
	if retrieved.ID != machine.ID {
		t.Errorf("Expected ID %s, got %s", machine.ID, retrieved.ID)
	}
	if retrieved.Name != machine.Name {
		t.Errorf("Expected Name %s, got %s", machine.Name, retrieved.Name)
	}

	// Test Get non-existing
	_, found = c.Get("non-existent")
	if found {
		t.Error("Expected not to find non-existent machine")
	}
}

func TestExpiration(t *testing.T) {
	ttl := 50 * time.Millisecond
	c := NewMetricsCache(ttl)

	machine := models.Machine{
		ID: "machine-expired",
	}

	c.Set(machine)

	// Verify it's there initially
	_, found := c.Get("machine-expired")
	if !found {
		t.Fatal("Machine should be in cache initially")
	}

	// Wait for expiration
	time.Sleep(ttl + 10*time.Millisecond)

	// Verify it's considered expired by Get
	_, found = c.Get("machine-expired")
	if found {
		t.Error("Machine should be expired and not returned by Get")
	}
}

func TestGetLastKnown(t *testing.T) {
	ttl := 50 * time.Millisecond
	c := NewMetricsCache(ttl)

	machine := models.Machine{
		ID: "machine-last-known",
	}

	c.Set(machine)

	// Wait for expiration
	time.Sleep(ttl + 10*time.Millisecond)

	// Verify Get returns nothing
	_, found := c.Get("machine-last-known")
	if found {
		t.Fatal("Get should return false for expired item")
	}

	// Verify GetLastKnown returns the item
	retrieved, found := c.GetLastKnown("machine-last-known")
	if !found {
		t.Error("GetLastKnown should return true even for expired item")
	}
	if retrieved.ID != machine.ID {
		t.Errorf("Expected ID %s, got %s", machine.ID, retrieved.ID)
	}
}

func TestInvalidate(t *testing.T) {
	c := NewMetricsCache(1 * time.Minute)

	machine := models.Machine{
		ID: "machine-invalidate",
	}

	c.Set(machine)

	// Ensure it's there
	if _, found := c.Get("machine-invalidate"); !found {
		t.Fatal("Machine not found after Set")
	}

	c.Invalidate("machine-invalidate")

	// Ensure it's gone
	if _, found := c.Get("machine-invalidate"); found {
		t.Error("Machine found after Invalidate")
	}

	// Ensure it's gone from GetLastKnown too (since Invalidate removes it from map)
	if _, found := c.GetLastKnown("machine-invalidate"); found {
		t.Error("Machine found in GetLastKnown after Invalidate")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewMetricsCache(1 * time.Minute)
	machineCount := 100
	var wg sync.WaitGroup

	// Concurrent Sets
	wg.Add(machineCount)
	for i := 0; i < machineCount; i++ {
		go func(id int) {
			defer wg.Done()
			machine := models.Machine{
				ID: "machine-concurrent",
			}
			c.Set(machine)
		}(i)
	}
	wg.Wait()

	// Concurrent Gets and Sets mixed
	wg.Add(machineCount * 2)
	for i := 0; i < machineCount; i++ {
		go func(id int) {
			defer wg.Done()
			c.Set(models.Machine{ID: "machine-concurrent"})
		}(i)
		go func(id int) {
			defer wg.Done()
			c.Get("machine-concurrent")
		}(i)
	}
	wg.Wait()

	// Just ensuring no panic occurred and basic operations worked
	_, found := c.Get("machine-concurrent")
	if !found {
		t.Error("Expected machine-concurrent to be present")
	}
}
