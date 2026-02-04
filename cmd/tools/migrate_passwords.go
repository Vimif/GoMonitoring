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
	generateKey := flag.Bool("generate-key", false, "GÃ©nÃ©rer une nouvelle master key")
	dryRun := flag.Bool("dry-run", false, "Afficher les changements sans les appliquer")
	flag.Parse()

	// Si demande de gÃ©nÃ©ration de master key
	if *generateKey {
		key, err := crypto.GenerateMasterKey()
		if err != nil {
			log.Fatalf("Erreur gÃ©nÃ©ration master key: %v", err)
		}
		fmt.Println("=== MASTER KEY GÃ‰NÃ‰RÃ‰E ===")
		fmt.Println("Copiez cette clÃ© dans votre variable d'environnement:")
		fmt.Printf("\nexport GO_MONITORING_MASTER_KEY=\"%s\"\n\n", key)
		fmt.Println("âš ï¸  IMPORTANT: Sauvegardez cette clÃ© en lieu sÃ»r!")
		fmt.Println("âš ï¸  Sans cette clÃ©, vous ne pourrez plus dÃ©chiffrer vos passwords!")
		return
	}

	// VÃ©rifier que la master key est configurÃ©e
	if os.Getenv(crypto.EnvMasterKey) == "" {
		log.Fatalf("Erreur: Variable d'environnement %s non dÃ©finie.\n"+
			"Utilisez --generate-key pour gÃ©nÃ©rer une nouvelle clÃ©.", crypto.EnvMasterKey)
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
			continue // Pas de password Ã  migrer
		}

		encrypted, wasMigrated, err := crypto.MigratePassword(machine.Password)
		if err != nil {
			log.Printf("âŒ Erreur migration password pour %s: %v", machine.ID, err)
			continue
		}

		if wasMigrated {
			if *dryRun {
				fmt.Printf("âœ“ [DRY-RUN] %s: Password serait chiffrÃ©\n", machine.ID)
			} else {
				machine.Password = encrypted
				fmt.Printf("âœ“ %s: Password chiffrÃ©\n", machine.ID)
			}
			migratedCount++
		} else {
			fmt.Printf("- %s: Password dÃ©jÃ  chiffrÃ©\n", machine.ID)
		}
	}

	// Sauvegarder si des changements ont Ã©tÃ© faits
	if migratedCount > 0 && !*dryRun {
		fmt.Printf("\nSauvegarde de %d password(s) chiffrÃ©(s)...\n", migratedCount)
		if err := config.SaveConfig(*configPath, cfg); err != nil {
			log.Fatalf("Erreur sauvegarde config: %v", err)
		}
		fmt.Println("âœ… Migration terminÃ©e avec succÃ¨s!")
	} else if *dryRun {
		fmt.Printf("\n[DRY-RUN] %d password(s) seraient migrÃ©s\n", migratedCount)
		fmt.Println("ExÃ©cutez sans --dry-run pour appliquer les changements")
	} else {
		fmt.Println("\nâœ… Aucun password Ã  migrer (tous dÃ©jÃ  chiffrÃ©s)")
	}
}

// loadConfigRaw charge la config sans dÃ©chiffrer (pour la migration)
func loadConfigRaw(path string) (*config.Config, error) {
	// Temporairement dÃ©sactiver la variable d'environnement pour Ã©viter le dÃ©chiffrement
	// lors du chargement initial (on veut migrer les passwords en clair)
	originalKey := os.Getenv(crypto.EnvMasterKey)
	os.Unsetenv(crypto.EnvMasterKey)

	cfg, err := config.LoadConfig(path)

	// Restaurer la master key
	os.Setenv(crypto.EnvMasterKey, originalKey)

	return cfg, err
}
