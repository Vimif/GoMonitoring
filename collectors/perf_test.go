package collectors

import (
	"testing"
	"time"

	"go-monitoring/models"
	"go-monitoring/ssh"

	"github.com/stretchr/testify/assert"
)

// SlowMockClient embeds ssh.MockClient and adds delay to Execute
type SlowMockClient struct {
	*ssh.MockClient
	Delay time.Duration
}

// Execute adds a delay before calling the embedded MockClient.Execute
func (m *SlowMockClient) Execute(cmd string) (string, error) {
	time.Sleep(m.Delay)
	return m.MockClient.Execute(cmd)
}

func TestCollectMetrics_Performance(t *testing.T) {
	// Create a machine
	machine := &models.Machine{
		ID:     "perf-test",
		OSType: "linux",
	}

	// Create a mock client with delay
	// We need about 5 calls: System, CPU, Memory, Disk, DiskIO
	// sequential execution takes ~1900ms due to multiple fallbacks in SystemInfo
    // when commands are missing in the mock.
    // SystemInfo alone takes ~900ms (due to 9 commands including fallbacks).
    // Parallel execution is limited by the longest task (SystemInfo), so ~900ms.
	mockClient := &SlowMockClient{
		MockClient: ssh.NewMockClientLinux(),
		Delay:      100 * time.Millisecond,
	}

    // Ensure we are connected
    mockClient.Connect()

	collector := &ConcurrentCollector{}

	start := time.Now()
	err := collector.collectMetrics(mockClient, machine)
	duration := time.Since(start)

	assert.NoError(t, err)

	// Expect parallel execution to be significantly faster than sequential (1900ms)
    // Should be around 900ms (longest single task)
	assert.Less(t, duration, 1200*time.Millisecond, "Parallel execution should be faster than 1200ms")

    t.Logf("Execution took %v", duration)
}
