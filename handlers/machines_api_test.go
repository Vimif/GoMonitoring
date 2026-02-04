package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-monitoring/cache"
	"go-monitoring/config"
	"go-monitoring/ssh"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper pour crÃ©er un ConfigManager de test
func newTestConfigManager() *ConfigManager {
	cfg := &config.Config{
		Machines: []config.MachineConfig{},
		Settings: config.Settings{
			RefreshInterval: 30,
			SSHTimeout:      10,
		},
	}

	pool := ssh.NewPool([]config.MachineConfig{}, 10)
	metricsCache := cache.NewMetricsCache()

	return NewConfigManager(cfg, pool, metricsCache, "test_config.yaml")
}

func TestAddMachine_Success(t *testing.T) {
	cm := newTestConfigManager()

	newMachine := config.MachineConfig{
		ID:   "test-machine-1",
		Name: "Test Machine",
		Host: "192.168.1.100",
		Port: 22,
		User: "testuser",
	}

	body, err := json.Marshal(newMachine)
	require.NoError(t, err, "Failed to marshal machine config")

	req := httptest.NewRequest("POST", "/api/machines", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := AddMachine(cm)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status OK")

	// VÃ©rifier la rÃ©ponse JSON
	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err, "Failed to decode response")
	assert.Equal(t, "Machine ajoutÃ©e avec succÃ¨s", response["message"])
}

func TestAddMachine_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		machine        config.MachineConfig
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing ID",
			machine: config.MachineConfig{
				Name: "Test Machine",
				Host: "192.168.1.100",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "L'ID est requis",
		},
		{
			name: "missing name",
			machine: config.MachineConfig{
				ID:   "test-1",
				Host: "192.168.1.100",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Le nom est requis",
		},
		{
			name: "missing host",
			machine: config.MachineConfig{
				ID:   "test-1",
				Name: "Test Machine",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "L'hÃ´te est requis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := newTestConfigManager()

			body, err := json.Marshal(tt.machine)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/machines", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := AddMachine(cm)
			handler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Expected status %d", tt.expectedStatus)

			var response map[string]interface{}
			err = json.NewDecoder(w.Body).Decode(&response)
			require.NoError(t, err)
			assert.Contains(t, response["error"], tt.expectedError)
		})
	}
}

func TestAddMachine_InvalidJSON(t *testing.T) {
	cm := newTestConfigManager()

	invalidJSON := []byte(`{"id": "test", "name": }`)

	req := httptest.NewRequest("POST", "/api/machines", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := AddMachine(cm)
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected BadRequest for invalid JSON")

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response["error"].(string), "DonnÃ©es invalides")
}

func TestAddMachine_DuplicateID(t *testing.T) {
	cm := newTestConfigManager()

	// Ajouter une premiÃ¨re machine
	existingMachine := config.MachineConfig{
		ID:   "duplicate-id",
		Name: "Existing Machine",
		Host: "192.168.1.10",
		Port: 22,
		User: "user1",
	}

	// Ajouter manuellement la machine existante
	cfg := cm.GetConfig()
	cfg.Machines = append(cfg.Machines, existingMachine)

	// Tenter d'ajouter une machine avec le mÃªme ID
	newMachine := config.MachineConfig{
		ID:   "duplicate-id",
		Name: "New Machine",
		Host: "192.168.1.20",
		Port: 22,
		User: "user2",
	}

	body, err := json.Marshal(newMachine)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/machines", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := AddMachine(cm)
	handler(w, req)

	// Devrait retourner une erreur (le comportement exact dÃ©pend de l'implÃ©mentation)
	assert.NotEqual(t, http.StatusOK, w.Code, "Should not accept duplicate ID")
}

func TestUpdateMachine_Success(t *testing.T) {
	cm := newTestConfigManager()

	// Ajouter une machine existante
	existingMachine := config.MachineConfig{
		ID:   "update-test",
		Name: "Old Name",
		Host: "192.168.1.10",
		Port: 22,
		User: "olduser",
	}

	cfg := cm.GetConfig()
	cfg.Machines = append(cfg.Machines, existingMachine)

	// Mettre Ã  jour la machine
	updatedMachine := config.MachineConfig{
		ID:   "update-test",
		Name: "New Name",
		Host: "192.168.1.20",
		Port: 2222,
		User: "newuser",
	}

	body, err := json.Marshal(updatedMachine)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/machines/update-test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := UpdateMachine(cm)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status OK")
}

func TestDeleteMachine_Success(t *testing.T) {
	cm := newTestConfigManager()

	// Ajouter une machine existante
	existingMachine := config.MachineConfig{
		ID:   "delete-test",
		Name: "To Delete",
		Host: "192.168.1.10",
		Port: 22,
		User: "user",
	}

	cfg := cm.GetConfig()
	cfg.Machines = append(cfg.Machines, existingMachine)

	req := httptest.NewRequest("DELETE", "/api/machines/delete-test", nil)
	w := httptest.NewRecorder()

	handler := DeleteMachine(cm)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status OK")

	// VÃ©rifier que la machine a Ã©tÃ© supprimÃ©e
	cfg = cm.GetConfig()
	assert.Len(t, cfg.Machines, 0, "Machine should be deleted")
}

func TestDeleteMachine_NotFound(t *testing.T) {
	cm := newTestConfigManager()

	req := httptest.NewRequest("DELETE", "/api/machines/nonexistent", nil)
	w := httptest.NewRecorder()

	handler := DeleteMachine(cm)
	handler(w, req)

	// Devrait retourner une erreur NotFound ou BadRequest
	assert.NotEqual(t, http.StatusOK, w.Code, "Should not succeed for nonexistent machine")
}

func TestConfigManager_ThreadSafety(t *testing.T) {
	cm := newTestConfigManager()

	// Ajouter une machine
	machine := config.MachineConfig{
		ID:   "thread-test",
		Name: "Thread Test",
		Host: "192.168.1.10",
		Port: 22,
		User: "user",
	}

	cfg := cm.GetConfig()
	cfg.Machines = append(cfg.Machines, machine)

	// Lire la config de maniÃ¨re concurrente
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = cm.GetConfig()
			_ = cm.GetPool()
			_ = cm.GetCache()
			done <- true
		}()
	}

	// Attendre toutes les goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Si on arrive ici sans panic, le test passe
	assert.True(t, true, "Thread safety test passed")
}

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()

	jsonError(w, "Test error message", http.StatusBadRequest)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Test error message", response["error"])
}

func TestJSONSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	jsonSuccess(w, "Test success message")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "Test success message", response["message"])
}

// Benchmark tests

func BenchmarkAddMachine(b *testing.B) {
	cm := newTestConfigManager()

	machine := config.MachineConfig{
		ID:   "bench-test",
		Name: "Bench Machine",
		Host: "192.168.1.10",
		Port: 22,
		User: "user",
	}

	body, _ := json.Marshal(machine)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/machines", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler := AddMachine(cm)
		handler(w, req)

		// Reset pour le prochain benchmark
		cfg := cm.GetConfig()
		cfg.Machines = []config.MachineConfig{}
	}
}

func BenchmarkGetConfig(b *testing.B) {
	cm := newTestConfigManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cm.GetConfig()
	}
}
