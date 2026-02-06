package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-monitoring/auth"
	"go-monitoring/cache"
	"go-monitoring/config"
	"go-monitoring/handlers"
	"go-monitoring/middleware"
	"go-monitoring/ssh"
	"go-monitoring/storage"
)

const configPath = "config.yaml"

func main() {
	// Charger la configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Erreur chargement configuration: %v", err)
	}

	log.Printf("Configuration chargée: %d machine(s)", len(cfg.Machines))

	// Créer le pool de connexions SSH
	pool := ssh.NewPool(cfg.Machines, cfg.Settings.SSHTimeout)
	defer pool.CloseAll()

	// Créer le cache de métriques (TTL 10 secondes)
	metricsCache := cache.NewMetricsCache(10 * time.Second)

	// Initialiser la base de données
	db, err := storage.InitDB("monitoring.db")
	if err != nil {
		log.Fatalf("Erreur initialisation base de données: %v", err)
	}

	// Démarrer la routine de nettoyage des tokens CSRF
	middleware.StartCleanupRoutine()
	log.Println("Routine de nettoyage CSRF démarrée")

	// Démarrer le Hub WebSocket
	go handlers.WSHub.Run()

	// Créer le gestionnaire de configuration (Thread-Safe)
	cm := handlers.NewConfigManager(cfg, pool, metricsCache, configPath)

	// Tâche de fond pour la collecte périodique (Historique - 1 min)
	go func() {
		log.Println("Démarrage de la collecte d'historique (tout les 1 minute)")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			// Récupérer la configuration et le pool à jour
			currentCfg, currentPool, _ := cm.GetConfigPoolAndCache()

			// Collecter les métriques pour l'historique
			machines := handlers.CollectAllMachines(currentCfg, currentPool, metricsCache, 20*time.Second, true)

			for _, m := range machines {
				if err := db.SaveMetric(m); err != nil {
					log.Printf("Erreur sauvegarde historique %s: %v", m.ID, err)
				}
			}
			// Nettoyage historique > 7 jours
			if err := db.CleanupOldMetrics(7 * 24 * time.Hour); err != nil {
				log.Printf("Erreur nettoyage historique: %v", err)
			}
		}
	}()

	// Tâche de fond pour le temps réel (WebSocket - 5s)
	go func() {
		log.Println("Démarrage de la collecte temps réel (tout les 5 secondes)")
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			// Récupérer la configuration et le pool à jour
			currentCfg, currentPool, _ := cm.GetConfigPoolAndCache()

			// Force refresh pour le temps réel
			machines := handlers.CollectAllMachines(currentCfg, currentPool, metricsCache, 5*time.Second, true)
			handlers.WSHub.Broadcast(machines)
		}
	}()

	// Créer le gestionnaire d'authentification
	userManager := auth.NewUserManager(db, cfg.Users)
	authManager := auth.NewAuthManager(userManager)

	// Lier le UserManager au ConfigManager (pour usage API)
	cm.SetUserManager(userManager)

	// Configurer le routeur
	mux := http.NewServeMux()

	// Routes d'authentification
	mux.HandleFunc("GET /login", authManager.LoginHandler)
	mux.HandleFunc("POST /login", authManager.LoginHandler)
	mux.HandleFunc("/logout", authManager.LogoutHandler)

	// Pages protégées
	log.Println("Registering GET /{$}")
	mux.HandleFunc("GET /{$}", authManager.Middleware(handlers.DashboardWithCM(cm, authManager)))

	log.Println("Registering GET /machine/{id}")
	mux.HandleFunc("GET /machine/{id}", authManager.Middleware(handlers.MachineDetailWithCM(cm, authManager)))

	mux.HandleFunc("GET /alerts", authManager.Middleware(handlers.RenderPageWithCM(cm, authManager, "alerts")))
	mux.HandleFunc("GET /settings", authManager.Middleware(handlers.RenderPageWithCM(cm, authManager, "settings")))
	mux.HandleFunc("GET /users", authManager.Middleware(handlers.UsersPage(cfg, authManager)))
	mux.HandleFunc("GET /audit", authManager.Middleware(handlers.AuditPage(cfg, db, authManager)))

	// API protégées
	mux.HandleFunc("GET /api/machine/{id}/disks", authManager.Middleware(handlers.DiskListWithCM(cm)))
	mux.HandleFunc("GET /api/machine/{id}/disk", authManager.Middleware(handlers.DiskDetailsWithCM(cm)))
	mux.HandleFunc("GET /api/machine/{id}/browse", authManager.Middleware(handlers.BrowseDirectoryWithCM(cm)))
	mux.HandleFunc("GET /api/machines", authManager.Middleware(handlers.ListMachines(cm)))
	mux.HandleFunc("POST /api/machines", authManager.Middleware(handlers.AddMachine(cm)))
	mux.HandleFunc("PUT /api/machines/{id}", authManager.Middleware(handlers.UpdateMachine(cm)))
	mux.HandleFunc("DELETE /api/machines/{id}", authManager.Middleware(handlers.RemoveMachine(cm)))
	mux.HandleFunc("GET /api/machine/{id}/history", authManager.Middleware(handlers.GetMachineHistory(db)))
	mux.HandleFunc("GET /api/machine/{id}/terminal", authManager.Middleware(handlers.WebTerminalHandler(cm)))
	mux.HandleFunc("GET /api/status", authManager.Middleware(handlers.GetStatus(cfg, pool, metricsCache)))
	mux.HandleFunc("POST /api/machine/{id}/service/{service}/{action}", authManager.Middleware(handlers.HandleServiceAction(cm, db, authManager)))
	mux.HandleFunc("GET /api/machine/{id}/logs", authManager.Middleware(handlers.ListLogSources(cm)))
	mux.HandleFunc("GET /api/machine/{id}/logs/view", authManager.Middleware(handlers.GetLogContent(cm, db, authManager)))

	// API Utilisateurs (Admin seulement)
	mux.HandleFunc("GET /api/users", authManager.Middleware(handlers.ListUsers(cm, authManager)))
	mux.HandleFunc("POST /api/users", authManager.Middleware(handlers.CreateUser(cm, authManager)))
	mux.HandleFunc("DELETE /api/users/{username}", authManager.Middleware(handlers.DeleteUser(cm, authManager)))
	mux.HandleFunc("PUT /api/users/{username}/password", authManager.Middleware(handlers.UpdateUserPassword(cm, authManager)))
	mux.HandleFunc("PUT /api/users/{username}/role", authManager.Middleware(handlers.UpdateUserRole(cm, authManager)))
	mux.HandleFunc("POST /api/users/{username}/toggle-status", authManager.Middleware(handlers.ToggleUserStatus(cm, authManager)))
	mux.HandleFunc("POST /api/users/{username}/unlock", authManager.Middleware(handlers.UnlockUser(cm, authManager)))
	mux.HandleFunc("POST /api/profile/password", authManager.Middleware(handlers.UpdateSelfPassword(cm, authManager)))

	// Fichiers statiques (publics)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Gestion de l'arrêt gracieux
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Arrêt du serveur...")
		pool.CloseAll()
		os.Exit(0)
	}()

	// Wrapper le serveur avec le middleware CSRF
	handler := middleware.CSRFMiddleware(mux)

	// Démarrer le serveur
	addr := ":8080"
	log.Printf("Serveur démarré sur http://localhost%s (Protection CSRF activée)", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Erreur serveur: %v", err)
	}
}
