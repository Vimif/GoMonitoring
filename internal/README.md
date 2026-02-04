# Architecture Interne

Cette structure suit les principes de Clean Architecture pour amÃ©liorer la maintenabilitÃ© et la testabilitÃ©.

## Structure

```
internal/
â”œâ”€â”€ domain/          # EntitÃ©s mÃ©tier (models)
â”‚   â”œâ”€â”€ machine.go   # DÃ©finitions des machines
â”‚   â”œâ”€â”€ user.go      # DÃ©finitions des utilisateurs
â”‚   â””â”€â”€ metric.go    # DÃ©finitions des mÃ©triques
â”œâ”€â”€ service/         # Logique mÃ©tier
â”‚   â”œâ”€â”€ machine_service.go      # Gestion des machines
â”‚   â”œâ”€â”€ monitoring_service.go   # Collecte et monitoring
â”‚   â””â”€â”€ user_service.go         # Gestion des utilisateurs
â””â”€â”€ repository/      # AccÃ¨s aux donnÃ©es
    â”œâ”€â”€ machine_repo.go         # Persistence des machines
    â”œâ”€â”€ user_repo.go            # Persistence des utilisateurs
    â””â”€â”€ metric_repo.go          # Persistence des mÃ©triques
```

## Principes

### Domain (EntitÃ©s MÃ©tier)
- Structures de donnÃ©es pure sans logique
- Pas de dÃ©pendances externes
- ReprÃ©sentent les concepts mÃ©tier

### Service (Logique MÃ©tier)
- ImplÃ©mentent les cas d'usage
- Utilisent les repositories via interfaces
- Orchestrent les opÃ©rations mÃ©tier
- IndÃ©pendants de la couche prÃ©sentation

### Repository (AccÃ¨s DonnÃ©es)
- Abstraient la persistance
- ImplÃ©mentent les interfaces dÃ©finies dans pkg/interfaces
- Peuvent utiliser SQL, fichiers, APIs externes

## DÃ©pendances

```
Handlers (HTTP)
    â†“
Services (Business Logic)
    â†“
Repositories (Data Access)
    â†“
Database / Config Files
```

## Migration

Cette structure remplace progressivement:
- `storage/` â†’ `internal/repository/`
- `models/` â†’ `internal/domain/`
- Logique Ã©parpillÃ©e dans handlers â†’ `internal/service/`

## Tests

Chaque couche a ses propres tests:
- `domain/*_test.go` - Tests unitaires simples
- `service/*_test.go` - Tests avec mocks de repositories
- `repository/*_test.go` - Tests d'intÃ©gration avec DB
