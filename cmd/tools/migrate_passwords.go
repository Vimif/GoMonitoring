package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go-monitoring/config"
	"go-monitoring/pkg/crypto"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Chemin vers le fichier de configuration")
	generateKey := flag.Bool("generate-key", false, "Générer une nouvelle master key")
	dryRun := flag.Bool("dry-run", false, "Afficher les changements sans les appliquer")
	flag.Parse()

	// Si demande de génération de master key
	if *generateKey {
		key, err := crypto.GenerateMasterKey()
		if err != nil {
			log.Fatalf("Erreur génération master key: %v", err)
		}
		fmt.Println("=== MASTER KEY GÉNÉRÉE ===")
		fmt.Println("Copiez cette clé dans votre variable d'environnement:")
		fmt.Printf("\nexport GO_MONITORING_MASTER_KEY=\"%s\"\n\n", key)
		fmt.Println("⚠ï¸  IMPORTANT: Sauvegardez cette clé en lieu sûr!")
		fmt.Println("⚠ï¸  Sans cette clé, vous ne pourrez plus déchiffrer vos passwords!")
		return
	}

	// Vérifier que la master key est configurée
	if os.Getenv(crypto.EnvMasterKey) == "" {
		log.Fatalf("Erreur: Variable d'environnement %s non définie.\n"+
			"Utilisez --generate-key pour générer une nouvelle clé.", crypto.EnvMasterKey)
	}

	// Charger la configuration
	fmt.Printf("Chargement de %s...\n", *configPath)
	cfg, err := loadConfigRaw(*configPath)
	if err != nil {
		log.Fatalf("Erreur chargement config: %v", err)
	}

	// Migrer les passwords
	migratedCount := 0
	for i := range cfg.Machines {
		machine := &cfg.Machines[i]

		if machine.Password == "" {
			continue // Pas de password à migrer
		}

		encrypted, wasMigrated, err := crypto.MigratePassword(machine.Password)
		if err != nil {
			log.Printf("âŒ Erreur migration password pour %s: %v", machine.ID, err)
			continue
		}

		if wasMigrated {
			if *dryRun {
				fmt.Printf("✓ [DRY-RUN] %s: Password serait chiffré\n", machine.ID)
			} else {
				machine.Password = encrypted
				fmt.Printf("✓ %s: Password chiffré\n", machine.ID)
			}
			migratedCount++
		} else {
			fmt.Printf("- %s: Password déjà chiffré\n", machine.ID)
		}
	}

	// Sauvegarder si des changements ont été faits
	if migratedCount > 0 && !*dryRun {
		fmt.Printf("\nSauvegarde de %d password(s) chiffré(s)...\n", migratedCount)
		if err := config.SaveConfig(*configPath, cfg); err != nil {
			log.Fatalf("Erreur sauvegarde config: %v", err)
		}
		fmt.Println("✅ Migration terminée avec succès!")
	} else if *dryRun {
		fmt.Printf("\n[DRY-RUN] %d password(s) seraient migrés\n", migratedCount)
		fmt.Println("Exécutez sans --dry-run pour appliquer les changements")
	} else {
		fmt.Println("\n✅ Aucun password à migrer (tous déjà chiffrés)")
	}
}

// loadConfigRaw charge la config sans déchiffrer (pour la migration)
func loadConfigRaw(path string) (*config.Config, error) {
	// Temporairement désactiver la variable d'environnement pour éviter le déchiffrement
	// lors du chargement initial (on veut migrer les passwords en clair)
	originalKey := os.Getenv(crypto.EnvMasterKey)
	os.Unsetenv(crypto.EnvMasterKey)

	cfg, err := config.LoadConfig(path)

	// Restaurer la master key
	os.Setenv(crypto.EnvMasterKey, originalKey)

	return cfg, err
}
