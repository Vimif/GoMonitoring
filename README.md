# Go Monitoring

Surveillance de machines Linux et Windows via SSH. Interface web avec mÃ©triques en temps rÃ©el, graphiques historiques et terminal SSH intÃ©grÃ©.

ConÃ§u pour fonctionner dans des environnements air-gapped (rÃ©seaux isolÃ©s sans accÃ¨s internet).

## Ce que Ã§a fait

- Surveille CPU, mÃ©moire, disques de vos machines Linux et Windows
- Affiche les mÃ©triques en temps rÃ©el via WebSocket (rafraÃ®chissement toutes les 5 secondes)
- Conserve 7 jours d'historique avec graphiques
- Terminal SSH directement dans le navigateur
- Explorateur de fichiers distant
- Gestion des services systemctl
- Support multi-utilisateurs avec rÃ´les (admin/viewer)

## Installation

**PrÃ©requis** : Go 1.21+, SQLite3, SSH activÃ© sur les machines Ã  surveiller.

```bash
# Cloner et installer
git clone https://github.com/votre-repo/go-monitoring.git
cd go-monitoring
go mod download

# GÃ©nÃ©rer une clÃ© de chiffrement pour les mots de passe SSH
go run cmd/tools/migrate_passwords.go --generate-key

# DÃ©finir la clÃ© (gardez-la en sÃ©curitÃ© !)
export GO_MONITORING_MASTER_KEY="votre-clÃ©-gÃ©nÃ©rÃ©e"

# Lancer
go run cmd/server/main.go
```

Ouvrir http://localhost:8080 - Login par dÃ©faut : `admin` / `admin`

## Configuration

Ã‰ditez `config.yaml` pour ajouter vos machines :

```yaml
settings:
  refresh_interval: 30   # secondes entre les collectes
  ssh_timeout: 10
  retention_days: 7

machines:
  - id: "serveur-web"
    name: "Serveur Web Prod"
    host: "192.168.1.10"
    port: 22
    user: "monitoring"
    group: "Production"
    os: "linux"
    key_path: "/home/user/.ssh/id_rsa"   # ou password: "..." (sera chiffrÃ© automatiquement)

  - id: "serveur-windows"
    name: "Windows Server"
    host: "192.168.1.20"
    port: 22
    user: "Administrator"
    group: "Infra"
    os: "windows"
    password: "motdepasse"
```

Pour gÃ©nÃ©rer un hash bcrypt (utilisateurs) :
```bash
go run cmd/tools/hash_gen.go -password "votremotdepasse"
```

## Structure du projet

```
cmd/server/main.go     Point d'entrÃ©e
cmd/tools/             Outils CLI (hash_gen, migrate_passwords)
internal/
  domain/              ModÃ¨les mÃ©tier (machine, user, metric)
  service/             Logique mÃ©tier
  repository/          AccÃ¨s aux donnÃ©es
pkg/
  crypto/              Chiffrement AES-256-GCM
  security/            Validation des commandes SSH
  ssh/                 Client SSH
handlers/              Routes HTTP
collectors/            Collecte des mÃ©triques via SSH
storage/               SQLite
templates/             Pages HTML
static/                CSS, JS, images
```

## Tests

```bash
go test ./...              # tous les tests
go test ./... -cover       # avec coverage
go test -race ./...        # dÃ©tection race conditions
```

Coverage actuel : ~80% sur le code critique.

## SÃ©curitÃ©

**Ce qui est en place :**
- Chiffrement AES-256-GCM des mots de passe SSH
- Protection CSRF sur toutes les requÃªtes
- Validation des commandes SSH (anti-injection)
- VÃ©rification des clÃ©s hÃ´tes SSH (TOFU)
- Mots de passe utilisateurs hashÃ©s avec bcrypt

**Ã€ ne jamais commiter :**
- `config.yaml` avec des vrais mots de passe
- Fichiers `.env`
- ClÃ©s SSH privÃ©es
- La base SQLite de production

## Routes principales

```
GET  /                                 Dashboard
GET  /machine/{id}                     DÃ©tail machine
GET  /api/machine/{id}/history         Historique mÃ©triques
GET  /api/machine/{id}/browse          Explorateur fichiers
GET  /api/machine/{id}/terminal        Terminal SSH (WebSocket)
POST /api/machines                     Ajouter machine
```

## DÃ©ploiement

Build optimisÃ© :
```bash
CGO_ENABLED=1 go build -ldflags="-s -w" -o monitoring cmd/server/main.go
```

Exemple de service systemd dans `/etc/systemd/system/monitoring.service` :
```ini
[Unit]
Description=Go Monitoring
After=network.target

[Service]
User=monitoring
WorkingDirectory=/opt/monitoring
Environment="GO_MONITORING_MASTER_KEY=votre-clÃ©"
ExecStart=/opt/monitoring/monitoring
Restart=always

[Install]
WantedBy=multi-user.target
```

## Contribution

Voir [CLAUDE.md](CLAUDE.md) pour les dÃ©tails techniques. Commentaires en franÃ§ais dans le code.

## Licence

MIT
