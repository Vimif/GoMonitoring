# Go Monitoring

Surveillance de machines Linux et Windows via SSH. Interface web avec métriques en temps réel, graphiques historiques et terminal SSH intégré.

Conçu pour fonctionner dans des environnements air-gapped (réseaux isolés sans accès internet).

## Ce que ça fait

- Surveille CPU, mémoire, disques de vos machines Linux et Windows
- Affiche les métriques en temps réel via WebSocket (rafraîchissement toutes les 5 secondes)
- Conserve 7 jours d'historique avec graphiques
- Terminal SSH directement dans le navigateur
- Explorateur de fichiers distant
- Gestion des services systemctl
- Support multi-utilisateurs avec rôles (admin/viewer)

## Installation

**Prérequis** : Go 1.21+, SQLite3, SSH activé sur les machines à surveiller.

```bash
# Cloner et installer
git clone https://github.com/votre-repo/go-monitoring.git
cd go-monitoring
go mod download

# Générer une clé de chiffrement pour les mots de passe SSH
go run cmd/tools/migrate_passwords.go --generate-key

# Définir la clé (gardez-la en sécurité !)
export GO_MONITORING_MASTER_KEY="votre-clé-générée"

# Lancer
go run cmd/server/main.go
```

Ouvrir http://localhost:8080 - Login par défaut : `admin` / `admin`

## Configuration

Éditez `config.yaml` pour ajouter vos machines :

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
    key_path: "/home/user/.ssh/id_rsa"   # ou password: "..." (sera chiffré automatiquement)

  - id: "serveur-windows"
    name: "Windows Server"
    host: "192.168.1.20"
    port: 22
    user: "Administrator"
    group: "Infra"
    os: "windows"
    password: "motdepasse"
```

Pour générer un hash bcrypt (utilisateurs) :
```bash
go run cmd/tools/hash_gen.go -password "votremotdepasse"
```

## Structure du projet

```
cmd/server/main.go     Point d'entrée
cmd/tools/             Outils CLI (hash_gen, migrate_passwords)
internal/
  domain/              Modèles métier (machine, user, metric)
  service/             Logique métier
  repository/          Accès aux données
pkg/
  crypto/              Chiffrement AES-256-GCM
  security/            Validation des commandes SSH
  ssh/                 Client SSH
handlers/              Routes HTTP
collectors/            Collecte des métriques via SSH
storage/               SQLite
templates/             Pages HTML
static/                CSS, JS, images
```

## Tests

```bash
go test ./...              # tous les tests
go test ./... -cover       # avec coverage
go test -race ./...        # détection race conditions
```

Coverage actuel : ~80% sur le code critique.

## Sécurité

**Ce qui est en place :**
- Chiffrement AES-256-GCM des mots de passe SSH
- Protection CSRF sur toutes les requêtes
- Validation des commandes SSH (anti-injection)
- Vérification des clés hôtes SSH (TOFU)
- Mots de passe utilisateurs hashés avec bcrypt

**À ne jamais commiter :**
- `config.yaml` avec des vrais mots de passe
- Fichiers `.env`
- Clés SSH privées
- La base SQLite de production

## Routes principales

```
GET  /                                 Dashboard
GET  /machine/{id}                     Détail machine
GET  /api/machine/{id}/history         Historique métriques
GET  /api/machine/{id}/browse          Explorateur fichiers
GET  /api/machine/{id}/terminal        Terminal SSH (WebSocket)
POST /api/machines                     Ajouter machine
```

## Déploiement

Build optimisé :
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
Environment="GO_MONITORING_MASTER_KEY=votre-clé"
ExecStart=/opt/monitoring/monitoring
Restart=always

[Install]
WantedBy=multi-user.target
```

## Contribution

Les contributions sont les bienvenues ! Forkez le projet, créez une branche, et soumettez une pull request. Commentaires en français dans le code.

## Licence

MIT
