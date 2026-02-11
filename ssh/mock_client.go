package ssh

import (
	"fmt"
	"sync"

	"go-monitoring/config"

	"golang.org/x/crypto/ssh"
)

// MockClient est un client SSH simulé pour les tests
type MockClient struct {
	// Commandes enregistrées et leurs réponses
	Commands map[string]string // cmd -> output
	Errors   map[string]error  // cmd -> error

	// Historique des commandes exécutées
	ExecutedCommands []string

	// État de la connexion
	connected bool

	// Config simulée
	config *config.MachineConfig

	mu sync.Mutex
}

// MockPool est un pool simulé pour les tests
type MockPool struct {
	clients map[string]*MockClient
	mu      sync.RWMutex
}

// NewMockClient crée un nouveau client SSH simulé
func NewMockClient(machineConfig *config.MachineConfig) *MockClient {
	return &MockClient{
		Commands:         make(map[string]string),
		Errors:           make(map[string]error),
		ExecutedCommands: []string{},
		connected:        false,
		config:           machineConfig,
	}
}

// NewMockPool crée un nouveau pool simulé
func NewMockPool() *MockPool {
	return &MockPool{
		clients: make(map[string]*MockClient),
	}
}

// AddClient ajoute un client simulé au pool
func (p *MockPool) AddClient(machineID string, client *MockClient) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[machineID] = client
}

// GetClient retourne un client simulé
func (p *MockPool) GetClient(machineID string) (*Client, error) {
	p.mu.RLock()
	mockClient, ok := p.clients[machineID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("machine non trouvée: %s", machineID)
	}

	// Retourner un wrapper qui implémente l'interface Client
	return &Client{
		config:  mockClient.config,
		timeout: 0,
	}, nil
}

// CloseAll ferme toutes les connexions du pool simulé
func (p *MockPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.Close()
	}
}

// SetResponse enregistre une réponse pour une commande
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

// SetResponseMap enregistre plusieurs réponses à la fois
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

	// Vérifier s'il y a une erreur de connexion enregistrée
	if err, exists := m.Errors["__connect__"]; exists {
		return err
	}

	m.connected = true
	return nil
}

// Execute simule l'exécution d'une commande SSH
func (m *MockClient) Execute(cmd string) (string, error) {
	// Simuler le comportement du vrai client qui tente de se connecter
	if err := m.Connect(); err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Enregistrer la commande dans l'historique
	m.ExecutedCommands = append(m.ExecutedCommands, cmd)

	// Vérifier si une erreur est enregistrée pour cette commande
	if err, exists := m.Errors[cmd]; exists {
		return "", err
	}

	// Retourner la réponse enregistrée
	if output, exists := m.Commands[cmd]; exists {
		return output, nil
	}

	// Si aucune réponse n'est enregistrée, retourner une erreur
	return "", fmt.Errorf("mock: no response registered for command: %s", cmd)
}

// NewSession simule la création d'une session SSH
func (m *MockClient) NewSession() (*ssh.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Pour les tests, on ne peut pas créer une vraie session SSH
	// Cette méthode devrait être mockée différemment si nécessaire
	return nil, fmt.Errorf("mock: NewSession not implemented")
}

// IsConnected retourne l'état de connexion simulé
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

// GetExecutedCommands retourne l'historique des commandes exécutées
func (m *MockClient) GetExecutedCommands() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Retourner une copie pour éviter les modifications externes
	commands := make([]string, len(m.ExecutedCommands))
	copy(commands, m.ExecutedCommands)
	return commands
}

// Reset réinitialise l'état du mock
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Commands = make(map[string]string)
	m.Errors = make(map[string]error)
	m.ExecutedCommands = []string{}
	m.connected = false
}

// --- Helpers pour créer des clients mock préconfigurés ---

// NewMockClientLinux crée un client mock configuré pour simuler un système Linux
func NewMockClientLinux() *MockClient {
	client := NewMockClient(&config.MachineConfig{
		ID:   "test-linux",
		Name: "Test Linux Machine",
		Host: "192.168.1.10",
		Port: 22,
		User: "test",
		OS:   "linux",
	})

	// Commandes par défaut pour Linux
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

// NewMockClientWindows crée un client mock configuré pour simuler un système Windows
func NewMockClientWindows() *MockClient {
	client := NewMockClient(&config.MachineConfig{
		ID:   "test-windows",
		Name: "Test Windows Machine",
		Host: "192.168.1.20",
		Port: 22,
		User: "test",
		OS:   "windows",
	})

	// Commandes par défaut pour Windows
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

// NewMockClientWithError crée un client mock qui retourne des erreurs
func NewMockClientWithError(errorMsg string) *MockClient {
	client := NewMockClient(&config.MachineConfig{
		ID:   "test-error",
		Name: "Test Error Machine",
		Host: "192.168.1.99",
		Port: 22,
		User: "test",
	})

	// Définir une erreur de connexion
	client.SetError("__connect__", fmt.Errorf(errorMsg))

	return client
}

// NewMockClientOffline crée un client mock qui simule une machine offline
func NewMockClientOffline() *MockClient {
	return NewMockClientWithError("connection refused: machine offline")
}

// NewMockClientTimeout crée un client mock qui simule un timeout
func NewMockClientTimeout() *MockClient {
	return NewMockClientWithError("i/o timeout: connection timed out")
}

// NewMockClientAuthFailed crée un client mock qui simule un échec d'authentification
func NewMockClientAuthFailed() *MockClient {
	return NewMockClientWithError("ssh: unable to authenticate: authentication failed")
}

// Vérifier que MockPool implémente ClientPool
var _ ClientPool = (*MockPool)(nil)
