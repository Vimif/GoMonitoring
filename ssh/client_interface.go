package ssh

import "golang.org/x/crypto/ssh"

// SSHExecutor définit l'interface minimale pour exécuter des commandes SSH
// Cette interface permet l'utilisation de mocks dans les tests
type SSHExecutor interface {
	Execute(cmd string) (string, error)
	Connect() error
	IsConnected() bool
	Close() error
	NewSession() (*ssh.Session, error)
}

// ClientPool définit l'interface pour un pool de clients SSH
type ClientPool interface {
	GetClient(machineID string) (*Client, error)
	CloseAll()
}

// Vérifier que Client implémente SSHExecutor
var _ SSHExecutor = (*Client)(nil)

// Vérifier que MockClient implémente SSHExecutor
var _ SSHExecutor = (*MockClient)(nil)

// Vérifier que Pool implémente ClientPool
var _ ClientPool = (*Pool)(nil)
