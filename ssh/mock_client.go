package ssh

import (
	"fmt"
	"sync"

	"go-monitoring/config"

	"golang.org/x/crypto/ssh"
)

// MockClient est un client SSH simulÃ© pour les tests
type MockClient struct {
	// Commandes enregistrÃ©es et leurs rÃ©ponses
	Commands map[string]string // cmd -> output
	Errors   map[string]error  // cmd -> error

	// Historique des commandes exÃ©cutÃ©es
	ExecutedCommands []string

	// Ã‰tat de la connexion
	connected bool

	// Config simulÃ©e
	config *config.MachineConfig

	mu sync.Mutex
}

// MockPool est un pool simulÃ© pour les tests
type MockPool struct {
	clients map[string]*MockClient
	mu      sync.RWMutex
}

// NewMockClient crÃ©e un nouveau client SSH simulÃ©
func NewMockClient(machineConfig *config.MachineConfig) *MockClient {
	return &MockClient{
		Commands:         make(map[string]string),
		Errors:           make(map[string]error),
		ExecutedCommands: []string{},
		connected:        false,
		config:           machineConfig,
	}
}

// NewMockPool crÃ©e un nouveau pool simulÃ©
func NewMockPool() *MockPool {
	return &MockPool{
		clients: make(map[string]*MockClient),
	}
}

// AddClient ajoute un client simulÃ© au pool
func (p *MockPool) AddClient(machineID string, client *MockClient) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[machineID] = client
}

// GetClient retourne un client simulÃ©
func (p *MockPool) GetClient(machineID string) (*Client, error) {
	p.mu.RLock()
	mockClient, ok := p.clients[machineID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("machine non trouvÃ©e: %s", machineID)
	}

	// Retourner un wrapper qui implÃ©mente l'interface Client
	return &Client{
		config:  mockClient.config,
		timeout: 0,
	}, nil
}

// SetResponse enregistre une rÃ©ponse pour une commande
func (m *MockClient) SetResponse(cmd, output string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Commands[cmd] = output
}

// SetError enregistre une erreur pour une commande
func (m *MockClient) SetError(cmd string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Errors[cmd] = err
}

// SetResponseMap enregistre plusieurs rÃ©ponses Ã  la fois
func (m *MockClient) SetResponseMap(responses map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for cmd, output := range responses {
		m.Commands[cmd] = output
	}
}

// Connect simule une connexion SSH
func (m *MockClient) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// VÃ©rifier s'il y a une erreur de connexion enregistrÃ©e
	if err, exists := m.Errors["__connect__"]; exists {
		return err
	}

	m.connected = true
	return nil
}

// Execute simule l'exÃ©cution d'une commande SSH
func (m *MockClient) Execute(cmd string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Enregistrer la commande dans l'historique
	m.ExecutedCommands = append(m.ExecutedCommands, cmd)

	// VÃ©rifier si une erreur est enregistrÃ©e pour cette commande
	if err, exists := m.Errors[cmd]; exists {
		return "", err
	}

	// Retourner la rÃ©ponse enregistrÃ©e
	if output, exists := m.Commands[cmd]; exists {
		return output, nil
	}

	// Si aucune rÃ©ponse n'est enregistrÃ©e, retourner une erreur
	return "", fmt.Errorf("mock: no response registered for command: %s", cmd)
}

// NewSession simule la crÃ©ation d'une session SSH
func (m *MockClient) NewSession() (*ssh.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Pour les tests, on ne peut pas crÃ©er une vraie session SSH
	// Cette mÃ©thode devrait Ãªtre mockÃ©e diffÃ©remment si nÃ©cessaire
	return nil, fmt.Errorf("mock: NewSession not implemented")
}

// IsConnected retourne l'Ã©tat de connexion simulÃ©
func (m *MockClient) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

// Close simule la fermeture de la connexion
func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

// GetExecutedCommands retourne l'historique des commandes exÃ©cutÃ©es
func (m *MockClient) GetExecutedCommands() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Retourner une copie pour Ã©viter les modifications externes
	commands := make([]string, len(m.ExecutedCommands))
	copy(commands, m.ExecutedCommands)
	return commands
}

// Reset rÃ©initialise l'Ã©tat du mock
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Commands = make(map[string]string)
	m.Errors = make(map[string]error)
	m.ExecutedCommands = []string{}
	m.connected = false
}

// --- Helpers pour crÃ©er des clients mock prÃ©configurÃ©s ---

// NewMockClientLinux crÃ©e un client mock configurÃ© pour simuler un systÃ¨me Linux
func NewMockClientLinux() *MockClient {
	client := NewMockClient(&config.MachineConfig{
		ID:   "test-linux",
		Name: "Test Linux Machine",
		Host: "192.168.1.10",
		Port: 22,
		User: "test",
		OS:   "linux",
	})

	// Commandes par dÃ©faut pour Linux
	client.SetResponseMap(map[string]string{
		"cat /proc/cpuinfo | grep 'model name' | head -1 | cut -d':' -f2": "Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz",
		"grep -c ^processor /proc/cpuinfo":                                 "8",
		"lscpu | grep '^CPU(s):' | awk '{print $2}'":                       "8",
		"cat /proc/cpuinfo | grep 'cpu MHz' | head -1 | cut -d':' -f2":     "3600.000",
		"top -bn1 | grep 'Cpu(s)' | head -1":                               "Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.7 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
		"free -b | grep Mem":                                               "Mem:   16777216000   8388608000   2097152000   104857600   6291456000   7340032000",
		"df -B1 -T | tail -n +2":                                           "/dev/sda1      ext4  107374182400  53687091200  48339148800   53% /",
		"systemctl is-active nginx apache2 mysql || true":                  "active\ninactive\nactive",
	})

	return client
}

// NewMockClientWindows crÃ©e un client mock configurÃ© pour simuler un systÃ¨me Windows
func NewMockClientWindows() *MockClient {
	client := NewMockClient(&config.MachineConfig{
		ID:   "test-windows",
		Name: "Test Windows Machine",
		Host: "192.168.1.20",
		Port: 22,
		User: "test",
		OS:   "windows",
	})

	// Commandes par dÃ©faut pour Windows
	client.SetResponseMap(map[string]string{
		`powershell -Command "(Get-CimInstance Win32_Processor).Name"`:                    "Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz",
		`powershell -Command "(Get-CimInstance Win32_Processor).NumberOfCores"`:           "8",
		`powershell -Command "(Get-CimInstance Win32_Processor).NumberOfLogicalProcessors"`: "8",
		`powershell -Command "(Get-CimInstance Win32_Processor).MaxClockSpeed"`:           "3600",
		`powershell -Command "(Get-CimInstance Win32_Processor).LoadPercentage"`:          "15.5",
		`powershell -Command "$os = Get-CimInstance Win32_OperatingSystem; Write-Output ('{0}|{1}' -f ($os.TotalVisibleMemorySize * 1KB), ($os.FreePhysicalMemory * 1KB))"`: "17179869184|5368709120",
		`powershell -Command "Get-CimInstance Win32_LogicalDisk | Where-Object {$_.DriveType -eq 3} | ForEach-Object { Write-Output ('{0}|{1}|{2}|{3}' -f $_.DeviceID, $_.Size, $_.FreeSpace, $_.MediaType) }"`: "C:|107374182400|53687091200|SSD",
	})

	return client
}

// NewMockClientWithError crÃ©e un client mock qui retourne des erreurs
func NewMockClientWithError(errorMsg string) *MockClient {
	client := NewMockClient(&config.MachineConfig{
		ID:   "test-error",
		Name: "Test Error Machine",
		Host: "192.168.1.99",
		Port: 22,
		User: "test",
	})

	// DÃ©finir une erreur de connexion
	client.SetError("__connect__", fmt.Errorf(errorMsg))

	return client
}

// NewMockClientOffline crÃ©e un client mock qui simule une machine offline
func NewMockClientOffline() *MockClient {
	return NewMockClientWithError("connection refused: machine offline")
}

// NewMockClientTimeout crÃ©e un client mock qui simule un timeout
func NewMockClientTimeout() *MockClient {
	return NewMockClientWithError("i/o timeout: connection timed out")
}

// NewMockClientAuthFailed crÃ©e un client mock qui simule un Ã©chec d'authentification
func NewMockClientAuthFailed() *MockClient {
	return NewMockClientWithError("ssh: unable to authenticate: authentication failed")
}
