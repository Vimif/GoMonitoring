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

// Vérifier que Client implémente SSHExecutor
var _ SSHExecutor = (*Client)(nil)

// Vérifier que MockClient implémente SSHExecutor
var _ SSHExecutor = (*MockClient)(nil)
