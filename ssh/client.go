package ssh

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"

	"go-monitoring/config"

	"golang.org/x/crypto/ssh"
)

// Client représente un client SSH vers une machine
type Client struct {
	config          *config.MachineConfig
	client          *ssh.Client
	timeout         time.Duration
	hostKeyCallback ssh.HostKeyCallback
	mu              sync.Mutex
}

// Pool gère les connexions SSH vers plusieurs machines
type Pool struct {
	clients        map[string]*Client
	timeout        time.Duration
	hostKeyManager *HostKeyManager
	mu             sync.RWMutex
}

// NewPool crée un nouveau pool de connexions SSH
func NewPool(machines []config.MachineConfig, timeout int) *Pool {
	// Initialiser le gestionnaire de host keys (mode TOFU par défaut)
	hostKeyManager, err := NewHostKeyManager("", true)
	if err != nil {
		// Fallback en mode insecure si échec (pour compatibilité)
		fmt.Fprintf(os.Stderr, "AVERTISSEMENT: Impossible d'initialiser HostKeyManager: %v\n", err)
		fmt.Fprintf(os.Stderr, "Mode INSECURE activé (vulnérable aux attaques MITM)\n")
	} else {
		fmt.Printf("ðŸ”’ Host Key Verification activée (TOFU mode)\n")
		fmt.Printf("   Known hosts: %s\n", hostKeyManager.GetKnownHostsPath())
	}

	pool := &Pool{
		clients:        make(map[string]*Client),
		timeout:        time.Duration(timeout) * time.Second,
		hostKeyManager: hostKeyManager,
	}

	// Définir le callback à utiliser
	var hostKeyCallback ssh.HostKeyCallback
	if hostKeyManager != nil {
		hostKeyCallback = hostKeyManager.HostKeyCallback()
	} else {
		// Fallback insecure
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	for i := range machines {
		pool.clients[machines[i].ID] = &Client{
			config:          &machines[i],
			timeout:         pool.timeout,
			hostKeyCallback: hostKeyCallback,
		}
	}

	return pool
}

// GetClient retourne un client SSH pour une machine
func (p *Pool) GetClient(machineID string) (*Client, error) {
	p.mu.RLock()
	client, ok := p.clients[machineID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("machine non trouvée: %s", machineID)
	}

	return client, nil
}

// Connect établit la connexion SSH
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked()
}

// connectLocked établit la connexion SSH en supposant que le lock est déjà acquis
func (c *Client) connectLocked() error {
	// Si déjà connecté, vérifier si la connexion est toujours valide
	if c.client != nil {
		_, _, err := c.client.SendRequest("keepalive", true, nil)
		if err == nil {
			return nil
		}
		c.client.Close()
		c.client = nil
	}

	// Préparer l'authentification
	var authMethods []ssh.AuthMethod

	// Authentification par clé SSH
	if c.config.KeyPath != "" {
		key, err := os.ReadFile(c.config.KeyPath)
		if err != nil {
			return fmt.Errorf("erreur lecture clé SSH: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("erreur parsing clé SSH: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Authentification par mot de passe
	if c.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.config.Password))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("aucune méthode d'authentification configurée")
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            authMethods,
		HostKeyCallback: c.hostKeyCallback,
		Timeout:         c.timeout,
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("erreur connexion SSH à %s: %w", addr, err)
	}

	c.client = client
	return nil
}

// Execute exécute une commande SSH et retourne la sortie
func (c *Client) Execute(cmd string) (string, error) {
	c.mu.Lock()
	// S'assurer que nous sommes connectés (sous lock)
	if err := c.connectLocked(); err != nil {
		c.mu.Unlock()
		return "", err
	}

	// Créer la session sous lock pour éviter les race conditions
	session, err := c.client.NewSession()
	c.mu.Unlock()

	if err != nil {
		return "", fmt.Errorf("erreur création session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("erreur exécution commande: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}

// NewSession crée une nouvelle session SSH interactive
func (c *Client) NewSession() (*ssh.Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.connectLocked(); err != nil {
		return nil, err
	}

	return c.client.NewSession()
}

// IsConnected vérifie si le client est connecté
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		return false
	}

	_, _, err := c.client.SendRequest("keepalive", true, nil)
	return err == nil
}

// Close ferme la connexion SSH
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// CloseAll ferme toutes les connexions du pool
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.Close()
	}
}
