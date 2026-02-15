package collectors

import (
	"fmt"
	"testing"

	"go-monitoring/ssh"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests d'intégration utilisant le mock SSH client

func TestCollectCPUInfo_Linux_Integration(t *testing.T) {
	// Créer un client mock Linux
	mockClient := ssh.NewMockClientLinux()

	// Tester la collecte CPU
	cpuInfo, err := CollectCPUInfo(mockClient, "linux")

	require.NoError(t, err, "CollectCPUInfo should not return error")
	assert.NotEmpty(t, cpuInfo.Model, "CPU model should not be empty")
	assert.Contains(t, cpuInfo.Model, "Intel", "CPU model should contain Intel")
	assert.Equal(t, 8, cpuInfo.Cores, "Should have 8 cores")
	assert.Equal(t, 8, cpuInfo.Threads, "Should have 8 threads")
	assert.InDelta(t, 3600.0, cpuInfo.MHz, 1.0, "CPU MHz should be around 3600")
	assert.InDelta(t, 7.3, cpuInfo.UsagePercent, 1.0, "CPU usage should be around 7.3%")

	// Vérifier que les bonnes commandes ont été exécutées
	executedCmds := mockClient.GetExecutedCommands()
	assert.NotEmpty(t, executedCmds, "Should have executed commands")
}

func TestCollectCPUInfo_Windows_Integration(t *testing.T) {
	// Créer un client mock Windows
	mockClient := ssh.NewMockClientWindows()

	// Tester la collecte CPU
	cpuInfo, err := CollectCPUInfo(mockClient, "windows")

	require.NoError(t, err, "CollectCPUInfo should not return error")
	assert.NotEmpty(t, cpuInfo.Model, "CPU model should not be empty")
	assert.Contains(t, cpuInfo.Model, "Intel", "CPU model should contain Intel")
	assert.Equal(t, 8, cpuInfo.Cores, "Should have 8 cores")
	assert.Equal(t, 8, cpuInfo.Threads, "Should have 8 threads")
	assert.InDelta(t, 3600.0, cpuInfo.MHz, 1.0, "CPU MHz should be around 3600")
	assert.InDelta(t, 15.5, cpuInfo.UsagePercent, 0.5, "CPU usage should be around 15.5%")
}

func TestCollectMemoryInfo_Linux_Integration(t *testing.T) {
	// Créer un client mock Linux
	mockClient := ssh.NewMockClientLinux()

	// Tester la collecte mémoire
	memInfo, err := CollectMemoryInfo(mockClient, "linux")

	require.NoError(t, err, "CollectMemoryInfo should not return error")
	assert.Greater(t, memInfo.Total, uint64(0), "Total memory should be greater than 0")
	assert.Greater(t, memInfo.Used, uint64(0), "Used memory should be greater than 0")
	assert.Greater(t, memInfo.Free, uint64(0), "Free memory should be greater than 0")
	assert.Greater(t, memInfo.Available, uint64(0), "Available memory should be greater than 0")
	assert.GreaterOrEqual(t, memInfo.UsedPercent, 0.0, "Used percent should be >= 0")
	assert.LessOrEqual(t, memInfo.UsedPercent, 100.0, "Used percent should be <= 100")

	// Vérifier les valeurs approximatives
	expectedTotal := uint64(16777216000) // ~16 GB
	assert.InDelta(t, float64(expectedTotal), float64(memInfo.Total), float64(expectedTotal)*0.01, "Total should be around 16GB")
}

func TestCollectMemoryInfo_Windows_Integration(t *testing.T) {
	// Créer un client mock Windows
	mockClient := ssh.NewMockClientWindows()

	// Tester la collecte mémoire
	memInfo, err := CollectMemoryInfo(mockClient, "windows")

	require.NoError(t, err, "CollectMemoryInfo should not return error")
	assert.Greater(t, memInfo.Total, uint64(0), "Total memory should be greater than 0")
	assert.Greater(t, memInfo.Used, uint64(0), "Used memory should be greater than 0")
	assert.Greater(t, memInfo.Free, uint64(0), "Free memory should be greater than 0")
	assert.GreaterOrEqual(t, memInfo.UsedPercent, 0.0, "Used percent should be >= 0")
	assert.LessOrEqual(t, memInfo.UsedPercent, 100.0, "Used percent should be <= 100")

	// Vérifier que Used = Total - Free
	assert.Equal(t, memInfo.Total-memInfo.Free, memInfo.Used, "Used should equal Total - Free")
}

func TestCollectDiskInfo_Linux_Integration(t *testing.T) {
	// Créer un client mock Linux
	mockClient := ssh.NewMockClientLinux()

	// Tester la collecte disque
	disks, err := CollectDiskInfo(mockClient, "linux")

	require.NoError(t, err, "CollectDiskInfo should not return error")
	assert.NotEmpty(t, disks, "Should have at least one disk")

	if len(disks) > 0 {
		disk := disks[0]
		assert.NotEmpty(t, disk.Device, "Disk device should not be empty")
		assert.NotEmpty(t, disk.FSType, "Filesystem type should not be empty")
		assert.NotEmpty(t, disk.MountPoint, "Mount point should not be empty")
		assert.Greater(t, disk.Total, uint64(0), "Total size should be greater than 0")
		assert.Greater(t, disk.Free, uint64(0), "Free space should be greater than 0")
		assert.GreaterOrEqual(t, disk.UsedPercent, 0.0, "Used percent should be >= 0")
		assert.LessOrEqual(t, disk.UsedPercent, 100.0, "Used percent should be <= 100")
	}
}

func TestCollectDiskInfo_Windows_Integration(t *testing.T) {
	// Créer un client mock Windows
	mockClient := ssh.NewMockClientWindows()

	// Tester la collecte disque
	disks, err := CollectDiskInfo(mockClient, "windows")

	require.NoError(t, err, "CollectDiskInfo should not return error")
	assert.NotEmpty(t, disks, "Should have at least one disk")

	if len(disks) > 0 {
		disk := disks[0]
		assert.Contains(t, disk.Device, ":", "Windows disk should contain :")
		assert.NotEmpty(t, disk.MountPoint, "Mount point should not be empty")
		assert.Greater(t, disk.Total, uint64(0), "Total size should be greater than 0")
	}
}

func TestCollectServices_Linux_Integration(t *testing.T) {
	// Créer un client mock Linux
	mockClient := ssh.NewMockClientLinux()

	// Tester la collecte de services
	services := []string{"nginx", "apache2", "mysql"}
	statuses, err := CollectServices(mockClient, services, "linux")

	require.NoError(t, err, "CollectServices should not return error")
	assert.Len(t, statuses, 3, "Should return status for 3 services")

	// Vérifier les statuts attendus
	assert.Equal(t, "nginx", statuses[0].Name)
	assert.Equal(t, "active", statuses[0].Status)

	assert.Equal(t, "apache2", statuses[1].Name)
	assert.Equal(t, "inactive", statuses[1].Status)

	assert.Equal(t, "mysql", statuses[2].Name)
	assert.Equal(t, "active", statuses[2].Status)
}

func TestCollectServices_EmptyList(t *testing.T) {
	mockClient := ssh.NewMockClientLinux()

	// Tester avec une liste vide
	statuses, err := CollectServices(mockClient, []string{}, "linux")

	require.NoError(t, err, "Should not return error for empty service list")
	assert.Empty(t, statuses, "Should return empty array")
}

// Tests avec erreurs simulées

func TestCollectCPUInfo_ConnectionError(t *testing.T) {
	// Créer un client mock qui simule une connexion échouée
	mockClient := ssh.NewMockClientOffline()

	// La collecte devrait échouer
	_, err := CollectCPUInfo(mockClient, "linux")
	assert.Error(t, err, "Should return error when connection fails")
	assert.Contains(t, err.Error(), "offline", "Error should mention offline")
}

func TestCollectMemoryInfo_CommandError(t *testing.T) {
	// Créer un client mock avec erreur sur commande spécifique
	mockClient := ssh.NewMockClientLinux()
	mockClient.SetError("free -b | grep Mem", fmt.Errorf("command not found"))

	// La collecte devrait échouer
	_, err := CollectMemoryInfo(mockClient, "linux")
	assert.Error(t, err, "Should return error when command fails")
}

func TestCollectDiskInfo_MalformedOutput(t *testing.T) {
	// Créer un client mock avec sortie malformée
	mockClient := ssh.NewMockClientLinux()
	mockClient.SetResponse("df -B1 -T | tail -n +2", "invalid output")

	// La collecte ne devrait pas échouer mais retourner un tableau vide
	disks, err := CollectDiskInfo(mockClient, "linux")
	require.NoError(t, err, "Should not return error for malformed output")
	assert.Empty(t, disks, "Should return empty array for malformed output")
}

// Tests de performance avec mock

func TestCollectMultipleMetrics_Performance(t *testing.T) {
	mockClient := ssh.NewMockClientLinux()

	// Collecter plusieurs métriques séquentiellement
	_, err1 := CollectCPUInfo(mockClient, "linux")
	_, err2 := CollectMemoryInfo(mockClient, "linux")
	_, err3 := CollectDiskInfo(mockClient, "linux")

	assert.NoError(t, err1, "CPU collection should succeed")
	assert.NoError(t, err2, "Memory collection should succeed")
	assert.NoError(t, err3, "Disk collection should succeed")

	// Vérifier qu'au moins 3 commandes ont été exécutées (CPU optimized from 5 to 1)
	executedCmds := mockClient.GetExecutedCommands()
	assert.GreaterOrEqual(t, len(executedCmds), 3, "Should have executed multiple commands")
}

// Tests de réinitialisation du mock

func TestMockClient_Reset(t *testing.T) {
	mockClient := ssh.NewMockClientLinux()

	// Exécuter quelques commandes
	_, _ = CollectCPUInfo(mockClient, "linux")
	assert.NotEmpty(t, mockClient.GetExecutedCommands(), "Should have executed commands")

	// Réinitialiser
	mockClient.Reset()

	// Vérifier que l'historique est vide
	assert.Empty(t, mockClient.GetExecutedCommands(), "Executed commands should be empty after reset")
	assert.False(t, mockClient.IsConnected(), "Should not be connected after reset")
}

// Benchmark avec mock client

func BenchmarkCollectCPUInfo_Mock(b *testing.B) {
	mockClient := ssh.NewMockClientLinux()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectCPUInfo(mockClient, "linux")
	}
}

func BenchmarkCollectMemoryInfo_Mock(b *testing.B) {
	mockClient := ssh.NewMockClientLinux()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectMemoryInfo(mockClient, "linux")
	}
}

func BenchmarkCollectDiskInfo_Mock(b *testing.B) {
	mockClient := ssh.NewMockClientLinux()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectDiskInfo(mockClient, "linux")
	}
}

func BenchmarkCollectAllMetrics_Mock(b *testing.B) {
	mockClient := ssh.NewMockClientLinux()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectCPUInfo(mockClient, "linux")
		CollectMemoryInfo(mockClient, "linux")
		CollectDiskInfo(mockClient, "linux")
	}
}
