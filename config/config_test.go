package config

import (
	"os"
	"testing"

	"go-monitoring/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveConfig_EncryptsPasswords(t *testing.T) {
	// Setup master key
	t.Setenv("GO_MONITORING_MASTER_KEY", "test-master-key-12345678901234567890123456789012")

	// Create temp file
	f, err := os.CreateTemp("", "config_test_*.yaml")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	f.Close()

	// Create config with plain text password
	plainPassword := "supersecret123"
	cfg := &Config{
		Machines: []MachineConfig{
			{
				ID:       "test-machine",
				Name:     "Test Machine",
				Host:     "localhost",
				User:     "user",
				Password: plainPassword,
			},
		},
	}

	// Save config
	err = SaveConfig(f.Name(), cfg)
	require.NoError(t, err)

	// Verify in-memory password is STILL plain text (crucial for SSH)
	assert.Equal(t, plainPassword, cfg.Machines[0].Password, "In-memory password should remain plain text")

	// Read file content
	content, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	contentStr := string(content)

	// Verify file content does NOT contain plain text password
	assert.NotContains(t, contentStr, plainPassword, "Saved config should not contain plain text password")

	// Verify file content contains encrypted password
	// We check if we can load it back and decrypt it
	loadedCfg, err := LoadConfig(f.Name())
	require.NoError(t, err)
	assert.Equal(t, plainPassword, loadedCfg.Machines[0].Password, "Loaded config should have decrypted password")
}

func TestAddMachine_KeepsPlainTextInMemory(t *testing.T) {
	// Setup master key
	t.Setenv("GO_MONITORING_MASTER_KEY", "test-master-key-12345678901234567890123456789012")

	cfg := &Config{
		Machines: []MachineConfig{},
	}

	plainPassword := "newpassword123"
	newMachine := MachineConfig{
		ID:       "new-machine",
		Name:     "New Machine",
		Host:     "192.168.1.100",
		User:     "admin",
		Password: plainPassword,
	}

	err := cfg.AddMachine(newMachine)
	require.NoError(t, err)

	// Verify added machine has plain text password in memory
	var addedMachine *MachineConfig
	for i := range cfg.Machines {
		if cfg.Machines[i].ID == "new-machine" {
			addedMachine = &cfg.Machines[i]
			break
		}
	}
	require.NotNil(t, addedMachine)

	// This assertion ensures the password remains plain text in memory (fixed behavior)
	assert.Equal(t, plainPassword, addedMachine.Password, "AddMachine should keep password plain text in memory")
	assert.False(t, crypto.IsEncrypted(addedMachine.Password), "AddMachine should not encrypt password in memory")
}

func TestUpdateMachine_KeepsPlainTextInMemory(t *testing.T) {
	// Setup master key
	t.Setenv("GO_MONITORING_MASTER_KEY", "test-master-key-12345678901234567890123456789012")

	cfg := &Config{
		Machines: []MachineConfig{
			{
				ID:       "update-machine",
				Name:     "Update Machine",
				Host:     "localhost",
				User:     "user",
				Password: "oldpassword",
			},
		},
	}

	plainPassword := "updatedpassword123"
	updatedMachine := MachineConfig{
		ID:       "update-machine",
		Name:     "Update Machine",
		Host:     "localhost",
		User:     "user",
		Password: plainPassword,
	}

	err := cfg.UpdateMachine(updatedMachine)
	require.NoError(t, err)

	// Verify updated machine has plain text password in memory
	assert.Equal(t, plainPassword, cfg.Machines[0].Password, "UpdateMachine should keep password plain text in memory")
	assert.False(t, crypto.IsEncrypted(cfg.Machines[0].Password), "UpdateMachine should not encrypt password in memory")
}
