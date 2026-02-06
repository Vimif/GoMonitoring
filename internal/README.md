# Architecture Interne

Cette structure suit les principes de Clean Architecture pour améliorer la maintenabilité et la testabilité.

## Structure

```
internal/
├── domain/          # Entités métier (models)
│   ├── machine.go   # Définitions des machines
│   ├── user.go      # Définitions des utilisateurs
│   └── metric.go    # Définitions des métriques
├── service/         # Logique métier
│   ├── machine_service.go      # Gestion des machines
│   ├── monitoring_service.go   # Collecte et monitoring
│   └── user_service.go         # Gestion des utilisateurs
└── repository/      # Accès aux données
    ├── machine_repo.go         # Persistence des machines
    ├── user_repo.go            # Persistence des utilisateurs
    └── metric_repo.go          # Persistence des métriques
```

## Principes

### Domain (Entités Métier)
- Structures de données pure sans logique
- Pas de dépendances externes
- Représentent les concepts métier

### Service (Logique Métier)
- Implémentent les cas d'usage
- Utilisent les repositories via interfaces
- Orchestrent les opérations métier
- Indépendants de la couche présentation

### Repository (Accès Données)
- Abstraient la persistance
- Implémentent les interfaces définies dans pkg/interfaces
- Peuvent utiliser SQL, fichiers, APIs externes

## Dépendances

```
Handlers (HTTP)
    ↓
Services (Business Logic)
    ↓
Repositories (Data Access)
    ↓
Database / Config Files
```

## Migration

Cette structure remplace progressivement:
- `storage/` → `internal/repository/`
- `models/` → `internal/domain/`
- Logique éparpillée dans handlers → `internal/service/`

## Tests

Chaque couche a ses propres tests:
- `domain/*_test.go` - Tests unitaires simples
- `service/*_test.go` - Tests avec mocks de repositories
- `repository/*_test.go` - Tests d'intégration avec DB
