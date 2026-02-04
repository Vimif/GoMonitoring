package ssh

import "golang.org/x/crypto/ssh"

// SSHExecutor dÃ©finit l'interface minimale pour exÃ©cuter des commandes SSH
// Cette interface permet l'utilisation de mocks dans les tests
type SSHExecutor interface {
	Execute(cmd string) (string, error)
	Connect() error
	IsConnected() bool
	Close() error
	NewSession() (*ssh.Session, error)
}

// VÃ©rifier que Client implÃ©mente SSHExecutor
var _ SSHExecutor = (*Client)(nil)

// VÃ©rifier que MockClient implÃ©mente SSHExecutor
var _ SSHExecutor = (*MockClient)(nil)
