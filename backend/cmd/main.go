package main

import (
	"log"
	"net/http"
	"os"

	"watchparty/backend/internal/database"
	"watchparty/backend/internal/middlewares"
	"watchparty/backend/internal/router"
)

func main() {
	// Connexion à MySQL
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Connexion DB échouée : %v", err)
	}
	defer db.Close()

	// Création des tables si elles n'existent pas
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Migration échouée : %v", err)
	}

	// Données de démonstration (films + user de test) si tables vides
	if err := database.Seed(db); err != nil {
		log.Fatalf("Seed échoué : %v", err)
	}

	// Construction du routeur (net/http natif, sans Gin)
	mux := router.New(db)

	// On enveloppe toutes les routes dans le middleware CORS
	handler := middlewares.CORS(mux)

	// Port configurable via la variable d'environnement PORT (défaut 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("Serveur démarré sur http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
