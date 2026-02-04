# CLAUDE.md

Guide pour Claude Code sur ce repository.

## Commandes

```bash
go build -o server.exe .    # Build
go run .                    # Lancer
go test ./...               # Tests
go test ./... -cover        # Tests avec coverage
```

Serveur sur http://localhost:8080

## Contexte

Application de monitoring pour machines Linux/Windows via SSH. Collecte CPU, mÃ©moire, disques et affiche dans un dashboard web temps rÃ©el.

**Air-gapped** : conÃ§u pour rÃ©seaux isolÃ©s, pas de dÃ©pendances CDN externes.

## Architecture

### Composants principaux

- **ConfigManager** (`handlers/machines_api.go`) : gÃ¨re config et pool SSH de faÃ§on thread-safe
- **SSH Pool** (`ssh/client.go`) : connexions lazy vers les machines
- **Collectors** (`collectors/`) : exÃ©cutent des commandes SSH et parsent la sortie
  - `system.go` : hostname, OS, uptime
  - `cpu.go` : info CPU (`/proc/cpuinfo` Linux, PowerShell Windows)
  - `memory.go` : RAM (`free -b` Linux, `Get-CimInstance` Windows)
  - `disk.go` : disques et navigation fichiers
- **Storage** (`storage/database.go`) : SQLite pour l'historique (7 jours par dÃ©faut)
- **Auth** (`auth/`) : sessions + bcrypt, user par dÃ©faut `admin:admin`

### Flux

1. `config.yaml` dÃ©finit les machines
2. Au dÃ©marrage, ConfigManager charge la config et crÃ©e le pool SSH
3. Dashboard collecte les mÃ©triques via SSH
4. Cache 10s + sauvegarde SQLite chaque minute
5. WebSocket push toutes les 5s

### Routes principales

```
GET  /                           Dashboard
GET  /machine/{id}               DÃ©tail machine
GET  /api/machine/{id}/history   Historique mÃ©triques
GET  /api/machine/{id}/browse    Explorateur fichiers
GET  /api/machine/{id}/terminal  Terminal SSH (WebSocket)
POST/PUT/DELETE /api/machines    CRUD machines
```

### Templates

Go `html/template` avec composition :
- `templates/layout/base.html` : layout avec sidebar
- `templates/dashboard.html` : cartes machines
- `templates/machine.html` : page dÃ©tail
- `templates/partials/` : composants rÃ©utilisables

Les fonctions custom doivent Ãªtre enregistrÃ©es avec `template.New().Funcs()` **avant** `ParseFiles()`.

## SÃ©curitÃ©

Tout est implÃ©mentÃ© :

- **CSRF** : `middleware/csrf.go` + `static/js/csrf.js` - tokens 24h liÃ©s aux sessions
- **Anti-injection** : `pkg/security/validation.go` - validation stricte des chemins et commandes
- **Chiffrement** : `pkg/crypto/encryption.go` - AES-256-GCM pour les mots de passe SSH
- **Host keys** : `ssh/hostkeys.go` - TOFU mode, stockage dans `~/.ssh/known_hosts_monitoring`
- **WebSocket CORS** : `handlers/websocket.go` - whitelist localhost

### ClÃ© maÃ®tre

```bash
# GÃ©nÃ©rer
go run cmd/tools/migrate_passwords.go --generate-key

# DÃ©finir (obligatoire pour chiffrement)
export GO_MONITORING_MASTER_KEY="votre-clÃ©"

# Migrer mots de passe existants
go run cmd/tools/migrate_passwords.go
```

### Ã€ ne pas commiter

- `config.yaml` avec vrais credentials
- `.env`
- ClÃ©s SSH privÃ©es
- Base SQLite de prod

## Tests

Coverage ~80% sur le code critique. Tests disponibles :
- `pkg/security/` : validation anti-injection
- `pkg/crypto/` : chiffrement
- `collectors/` : parsing Linux/Windows
- `handlers/` : API
- `ssh/mock_client.go` : mock pour tests sans connexion

## Structure

```
cmd/server/main.go          Point d'entrÃ©e
cmd/tools/                   hash_gen.go, migrate_passwords.go
internal/
  domain/                    ModÃ¨les (machine, user, metric)
  service/                   Logique mÃ©tier
  repository/                AccÃ¨s donnÃ©es
pkg/
  crypto/                    Chiffrement AES-256-GCM
  security/                  Validation commandes
  interfaces/                Interfaces services/repos
  logger/                    Logger structurÃ©
  contextutil/               Timeouts
handlers/                    Routes HTTP
collectors/                  Collecte mÃ©triques SSH
  concurrent_collector.go    Collection parallÃ¨le avec semaphore
storage/                     SQLite
templates/                   HTML
static/                      CSS, JS
frontend/                    Build Vite (ES6 modules)
```

## Configuration

```yaml
settings:
  refresh_interval: 30
  ssh_timeout: 10
  retention_days: 7

machines:
  - id: "unique-id"
    name: "Display Name"
    host: "192.168.1.10"
    port: 22
    user: "monitoring"
    group: "Production"
    key_path: "/path/to/id_rsa"  # ou password: "secret"
```

## Notes

- Commentaires en franÃ§ais dans le code
- Collection parallÃ¨le : `collectors/concurrent_collector.go` utilise semaphore + goroutines
- Les handlers n'utilisent pas encore les services de `internal/service/` (refactoring Ã  faire)
