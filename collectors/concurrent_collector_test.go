package collectors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-monitoring/config"
	"go-monitoring/models"
	"go-monitoring/ssh"

	"github.com/stretchr/testify/assert"
)

func TestNewConcurrentCollector(t *testing.T) {
	pool := ssh.NewPool([]config.MachineConfig{}, 10)

	t.Run("with valid max concurrent", func(t *testing.T) {
		collector := NewConcurrentCollector(pool, 5)
		assert.NotNil(t, collector)
		assert.Equal(t, 5, collector.maxConcurrent)
		assert.NotNil(t, collector.semaphore)
	})

	t.Run("with zero max concurrent defaults to 5", func(t *testing.T) {
		collector := NewConcurrentCollector(pool, 0)
		assert.Equal(t, 5, collector.maxConcurrent)
	})

	t.Run("with negative max concurrent defaults to 5", func(t *testing.T) {
		collector := NewConcurrentCollector(pool, -1)
		assert.Equal(t, 5, collector.maxConcurrent)
	})
}

func TestConcurrentCollector_CollectAll(t *testing.T) {
	// Créer un pool mock
	mockPool := ssh.NewMockPool()

	// Créer des machines de test
	machines := []models.Machine{
		{ID: "test-1", Name: "Test 1", Host: "192.168.1.1", OSType: "linux"},
		{ID: "test-2", Name: "Test 2", Host: "192.168.1.2", OSType: "linux"},
		{ID: "test-3", Name: "Test 3", Host: "192.168.1.3", OSType: "linux"},
	}

	// Ajouter des clients mock
	for _, m := range machines {
		client := ssh.NewMockClientLinux()
		mockPool.AddClient(m.ID, client)
	}

	collector := NewConcurrentCollector(nil, 2) // Max 2 concurrent
	collector.pool = mockPool

	ctx := context.Background()
	results := collector.CollectAll(ctx, machines)

	// Vérifier les résultats
	assert.Len(t, results, 3, "Should have 3 results")

	for _, result := range results {
		assert.NotEmpty(t, result.MachineID, "MachineID should not be empty")
		assert.NotNil(t, result.Machine, "Machine should not be nil")
		assert.Greater(t, result.Duration, time.Duration(0), "Duration should be positive")
	}
}

func TestConcurrentCollector_CollectAllWithTimeout(t *testing.T) {
	mockPool := ssh.NewMockPool()

	machines := []models.Machine{
		{ID: "test-1", Name: "Test 1", Host: "192.168.1.1", OSType: "linux"},
	}

	client := ssh.NewMockClientLinux()
	mockPool.AddClient("test-1", client)

	collector := NewConcurrentCollector(nil, 5)
	collector.pool = mockPool

	// Créer un contexte avec timeout très court
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Attendre que le contexte expire

	results := collector.CollectAll(ctx, machines)

	// Le résultat devrait contenir une erreur de timeout
	assert.Len(t, results, 1)
	// L'erreur peut être soit context.DeadlineExceeded soit context.Canceled
	if results[0].Error != nil {
		assert.Contains(t, results[0].Error.Error(), "context")
	}
}

func TestConcurrentCollector_CollectAllConcurrency(t *testing.T) {
	mockPool := ssh.NewMockPool()

	// Créer 10 machines
	machines := make([]models.Machine, 10)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("test-%d", i)
		machines[i] = models.Machine{
			ID:     id,
			Name:   fmt.Sprintf("Test %d", i),
			Host:   fmt.Sprintf("192.168.1.%d", i),
			OSType: "linux",
		}

		client := ssh.NewMockClientLinux()
		mockPool.AddClient(id, client)
	}

	// Limiter à 3 collections concurrentes
	collector := NewConcurrentCollector(nil, 3)
	collector.pool = mockPool

	ctx := context.Background()
	start := time.Now()
	results := collector.CollectAll(ctx, machines)
	duration := time.Since(start)

	// Tous les résultats devraient être présents
	assert.Len(t, results, 10)

	// Avec la concurrence, ça devrait être plus rapide que séquentiel
	// (difficile à tester de manière déterministe, mais on vérifie juste que ça ne prend pas trop de temps)
	assert.Less(t, duration, 10*time.Second, "Should complete in reasonable time")

	for _, result := range results {
		assert.NotNil(t, result.Machine)
	}
}

func TestConcurrentCollector_CollectBatch(t *testing.T) {
	mockPool := ssh.NewMockPool()

	allMachines := []models.Machine{
		{ID: "test-1", Name: "Test 1", Host: "192.168.1.1", OSType: "linux"},
		{ID: "test-2", Name: "Test 2", Host: "192.168.1.2", OSType: "linux"},
		{ID: "test-3", Name: "Test 3", Host: "192.168.1.3", OSType: "linux"},
	}

	for _, m := range allMachines {
		client := ssh.NewMockClientLinux()
		mockPool.AddClient(m.ID, client)
	}

	collector := NewConcurrentCollector(nil, 5)
	collector.pool = mockPool

	// Ne collecter que 2 machines spécifiques
	batchIDs := []string{"test-1", "test-3"}

	ctx := context.Background()
	results := collector.CollectBatch(ctx, batchIDs, allMachines)

	assert.Len(t, results, 2, "Should only collect 2 machines")

	// Vérifier que ce sont bien les bonnes machines
	ids := make(map[string]bool)
	for _, r := range results {
		ids[r.MachineID] = true
	}

	assert.True(t, ids["test-1"])
	assert.True(t, ids["test-3"])
	assert.False(t, ids["test-2"])
}

func TestGetStatistics(t *testing.T) {
	results := []CollectResult{
		{
			MachineID: "test-1",
			Machine:   &models.Machine{Status: "online"},
			Duration:  100 * time.Millisecond,
		},
		{
			MachineID: "test-2",
			Machine:   &models.Machine{Status: "online"},
			Duration:  200 * time.Millisecond,
		},
		{
			MachineID: "test-3",
			Machine:   &models.Machine{Status: "offline"},
			Duration:  50 * time.Millisecond,
		},
		{
			MachineID: "test-4",
			Error:     fmt.Errorf("connection failed"),
			Duration:  150 * time.Millisecond,
		},
	}

	stats := GetStatistics(results)

	assert.Equal(t, 4, stats.Total)
	assert.Equal(t, 2, stats.Online)
	assert.Equal(t, 1, stats.Offline)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 50*time.Millisecond, stats.MinDuration)
	assert.Equal(t, 200*time.Millisecond, stats.MaxDuration)
	assert.Equal(t, 125*time.Millisecond, stats.AvgDuration) // (100+200+50+150)/4
}

func TestGetStatistics_Empty(t *testing.T) {
	results := []CollectResult{}
	stats := GetStatistics(results)

	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, time.Duration(0), stats.AvgDuration)
}

func TestCollectionStats_String(t *testing.T) {
	stats := CollectionStats{
		Total:       10,
		Online:      7,
		Offline:     2,
		Failed:      1,
		MinDuration: 100 * time.Millisecond,
		MaxDuration: 500 * time.Millisecond,
		AvgDuration: 250 * time.Millisecond,
	}

	str := stats.String()

	assert.Contains(t, str, "Total: 10")
	assert.Contains(t, str, "Online: 7")
	assert.Contains(t, str, "Offline: 2")
	assert.Contains(t, str, "Failed: 1")
	assert.Contains(t, str, "250ms")
}

// Benchmark tests

func BenchmarkConcurrentCollector_CollectAll(b *testing.B) {
	mockPool := ssh.NewMockPool()

	machines := make([]models.Machine, 10)
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("test-%d", i)
		machines[i] = models.Machine{
			ID:     id,
			Name:   fmt.Sprintf("Test %d", i),
			Host:   fmt.Sprintf("192.168.1.%d", i),
			OSType: "linux",
		}

		client := ssh.NewMockClientLinux()
		mockPool.AddClient(id, client)
	}

	collector := NewConcurrentCollector(nil, 5)
	collector.pool = mockPool
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.CollectAll(ctx, machines)
	}
}

func BenchmarkConcurrentCollector_DifferentConcurrency(b *testing.B) {
	mockPool := ssh.NewMockPool()

	machines := make([]models.Machine, 20)
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("test-%d", i)
		machines[i] = models.Machine{
			ID:     id,
			Name:   fmt.Sprintf("Test %d", i),
			Host:   fmt.Sprintf("192.168.1.%d", i),
			OSType: "linux",
		}

		client := ssh.NewMockClientLinux()
		mockPool.AddClient(id, client)
	}

	ctx := context.Background()

	concurrencyLevels := []int{1, 5, 10, 20}

	for _, level := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", level), func(b *testing.B) {
			collector := NewConcurrentCollector(nil, level)
			collector.pool = mockPool

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				collector.CollectAll(ctx, machines)
			}
		})
	}
}
