package ssh

import (
	"fmt"
	"testing"

	"go-monitoring/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMockClient(t *testing.T) {
	mockConfig := &config.MachineConfig{
		ID:   "test-1",
		Name: "Test Machine",
		Host: "192.168.1.10",
		Port: 22,
		User: "testuser",
	}

	client := NewMockClient(mockConfig)

	assert.NotNil(t, client, "MockClient should not be nil")
	assert.NotNil(t, client.Commands, "Commands map should be initialized")
	assert.NotNil(t, client.Errors, "Errors map should be initialized")
	assert.Empty(t, client.ExecutedCommands, "Executed commands should be empty initially")
	assert.False(t, client.IsConnected(), "Should not be connected initially")
}

func TestMockClient_SetResponse(t *testing.T) {
	client := NewMockClient(nil)

	cmd := "echo hello"
	output := "hello"

	client.SetResponse(cmd, output)

	result, err := client.Execute(cmd)
	require.NoError(t, err, "Execute should not return error")
	assert.Equal(t, output, result, "Output should match set response")
}

func TestMockClient_SetError(t *testing.T) {
	client := NewMockClient(nil)

	cmd := "failing command"
	expectedErr := fmt.Errorf("command failed")

	client.SetError(cmd, expectedErr)

	_, err := client.Execute(cmd)
	require.Error(t, err, "Execute should return error")
	assert.Equal(t, expectedErr, err, "Error should match set error")
}

func TestMockClient_SetResponseMap(t *testing.T) {
	client := NewMockClient(nil)

	responses := map[string]string{
		"cmd1": "output1",
		"cmd2": "output2",
		"cmd3": "output3",
	}

	client.SetResponseMap(responses)

	for cmd, expectedOutput := range responses {
		output, err := client.Execute(cmd)
		require.NoError(t, err, "Execute should not return error for %s", cmd)
		assert.Equal(t, expectedOutput, output, "Output should match for %s", cmd)
	}
}

func TestMockClient_Connect(t *testing.T) {
	client := NewMockClient(nil)

	assert.False(t, client.IsConnected(), "Should not be connected initially")

	err := client.Connect()
	require.NoError(t, err, "Connect should not return error")
	assert.True(t, client.IsConnected(), "Should be connected after Connect()")
}

func TestMockClient_ConnectError(t *testing.T) {
	client := NewMockClient(nil)

	expectedErr := fmt.Errorf("connection refused")
	client.SetError("__connect__", expectedErr)

	err := client.Connect()
	require.Error(t, err, "Connect should return error")
	assert.Equal(t, expectedErr, err, "Error should match")
	assert.False(t, client.IsConnected(), "Should not be connected on error")
}

func TestMockClient_Close(t *testing.T) {
	client := NewMockClient(nil)

	client.Connect()
	assert.True(t, client.IsConnected(), "Should be connected")

	err := client.Close()
	require.NoError(t, err, "Close should not return error")
	assert.False(t, client.IsConnected(), "Should not be connected after Close()")
}

func TestMockClient_ExecuteHistory(t *testing.T) {
	client := NewMockClient(nil)

	commands := []string{"cmd1", "cmd2", "cmd3"}
	for _, cmd := range commands {
		client.SetResponse(cmd, "output")
		client.Execute(cmd)
	}

	history := client.GetExecutedCommands()
	assert.Len(t, history, 3, "Should have 3 commands in history")
	assert.Equal(t, commands, history, "History should match executed commands")
}

func TestMockClient_ExecuteNoResponse(t *testing.T) {
	client := NewMockClient(nil)

	_, err := client.Execute("unknown command")
	require.Error(t, err, "Execute should return error for unregistered command")
	assert.Contains(t, err.Error(), "no response registered", "Error should mention no response")
}

func TestMockClient_Reset(t *testing.T) {
	client := NewMockClient(nil)

	// Setup client with some state
	client.SetResponse("cmd1", "output1")
	client.SetError("cmd2", fmt.Errorf("error"))
	client.Connect()
	client.Execute("cmd1")

	assert.True(t, client.IsConnected(), "Should be connected")
	assert.NotEmpty(t, client.GetExecutedCommands(), "Should have executed commands")

	// Reset
	client.Reset()

	// Verify reset state
	assert.False(t, client.IsConnected(), "Should not be connected after reset")
	assert.Empty(t, client.GetExecutedCommands(), "Executed commands should be empty")
	assert.Empty(t, client.Commands, "Commands map should be empty")
	assert.Empty(t, client.Errors, "Errors map should be empty")
}

func TestNewMockClientLinux(t *testing.T) {
	client := NewMockClientLinux()

	assert.NotNil(t, client, "MockClient should not be nil")
	assert.Equal(t, "linux", client.config.OS, "OS should be linux")

	// Verify some default Linux commands are registered
	output, err := client.Execute("grep -c ^processor /proc/cpuinfo")
	require.NoError(t, err, "Should have response for processor count command")
	assert.NotEmpty(t, output, "Output should not be empty")

	output, err = client.Execute("free -b | grep Mem")
	require.NoError(t, err, "Should have response for memory command")
	assert.NotEmpty(t, output, "Output should not be empty")
}

func TestNewMockClientWindows(t *testing.T) {
	client := NewMockClientWindows()

	assert.NotNil(t, client, "MockClient should not be nil")
	assert.Equal(t, "windows", client.config.OS, "OS should be windows")

	// Verify some default Windows commands are registered
	output, err := client.Execute(`powershell -Command "(Get-CimInstance Win32_Processor).Name"`)
	require.NoError(t, err, "Should have response for CPU name command")
	assert.NotEmpty(t, output, "Output should not be empty")
}

func TestNewMockClientOffline(t *testing.T) {
	client := NewMockClientOffline()

	err := client.Connect()
	require.Error(t, err, "Connect should return error for offline client")
	assert.Contains(t, err.Error(), "offline", "Error should mention offline")
}

func TestNewMockClientTimeout(t *testing.T) {
	client := NewMockClientTimeout()

	err := client.Connect()
	require.Error(t, err, "Connect should return error for timeout client")
	assert.Contains(t, err.Error(), "timeout", "Error should mention timeout")
}

func TestNewMockClientAuthFailed(t *testing.T) {
	client := NewMockClientAuthFailed()

	err := client.Connect()
	require.Error(t, err, "Connect should return error for auth failed client")
	assert.Contains(t, err.Error(), "authentication", "Error should mention authentication")
}

func TestMockPool_AddClient(t *testing.T) {
	pool := NewMockPool()
	client := NewMockClientLinux()

	pool.AddClient("test-1", client)

	// Verify client was added
	assert.Len(t, pool.clients, 1, "Pool should have 1 client")
}

func TestMockPool_GetClient(t *testing.T) {
	pool := NewMockPool()
	mockClient := NewMockClientLinux()

	pool.AddClient("test-1", mockClient)

	// Get client (returns *Client wrapper, not *MockClient)
	client, err := pool.GetClient("test-1")
	require.NoError(t, err, "GetClient should not return error")
	assert.NotNil(t, client, "Client should not be nil")
}

func TestMockPool_GetClientNotFound(t *testing.T) {
	pool := NewMockPool()

	_, err := pool.GetClient("nonexistent")
	require.Error(t, err, "GetClient should return error for nonexistent machine")
	assert.Contains(t, err.Error(), "non trouvée", "Error should mention machine not found")
}

func TestMockClient_ConcurrentExecute(t *testing.T) {
	client := NewMockClient(nil)
	client.SetResponse("test cmd", "output")

	// Execute concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			client.Execute("test cmd")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all executions were recorded
	history := client.GetExecutedCommands()
	assert.Len(t, history, 10, "Should have 10 commands in history")
}

// Benchmark tests

func BenchmarkMockClient_Execute(b *testing.B) {
	client := NewMockClientLinux()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Execute("grep -c ^processor /proc/cpuinfo")
	}
}

func BenchmarkMockClient_SetResponse(b *testing.B) {
	client := NewMockClient(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.SetResponse("cmd", "output")
	}
}

func BenchmarkMockClient_GetExecutedCommands(b *testing.B) {
	client := NewMockClient(nil)
	client.SetResponse("cmd", "output")

	// Execute some commands
	for i := 0; i < 100; i++ {
		client.Execute("cmd")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetExecutedCommands()
	}
}
