package router

import (
	"database/sql"
	"net/http"

	"watchparty/backend/internal/controllers"
)

// New construit le multiplexeur HTTP avec toutes les routes de l'API.
// Utilise le routing natif de net/http (Go 1.22+) : méthode + path + {param}.
// Aucun framework externe (pas de Gin).
func New(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	// Santé / test de la chaîne
	mux.HandleFunc("GET /api/health", controllers.Health)

	// Authentification / utilisateurs
	mux.HandleFunc("POST /api/register", controllers.Register(db))
	mux.HandleFunc("POST /api/login", controllers.Login(db))
	mux.HandleFunc("GET /api/users", controllers.GetUsers(db))

	// Films
	mux.HandleFunc("GET /api/movies", controllers.GetMovies(db))
	mux.HandleFunc("POST /api/movies", controllers.CreateMovie(db))

	// WatchParties
	mux.HandleFunc("GET /api/parties", controllers.GetParties(db))
	mux.HandleFunc("POST /api/parties", controllers.CreateParty(db))
	mux.HandleFunc("GET /api/parties/{id}", controllers.GetPartyByID(db))

	// Swipes (cœur du projet)
	mux.HandleFunc("POST /api/parties/{id}/swipes", controllers.CreateSwipe(db))
	mux.HandleFunc("GET /api/parties/{id}/swipes", controllers.GetSwipes(db))

	// Recommandation
	mux.HandleFunc("POST /api/parties/{id}/recommendation/generate", controllers.GenerateRecommendation(db))
	mux.HandleFunc("GET /api/parties/{id}/recommendation", controllers.GetRecommendation(db))

	return mux
}
