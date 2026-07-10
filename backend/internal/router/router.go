package router

import (
	"database/sql"
	"net/http"

	"watchparty/backend/internal/controllers"
	"watchparty/backend/internal/middlewares"
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
	mux.HandleFunc("GET /api/me", middlewares.RequireAuth(controllers.GetMe(db)))

	// Films
	mux.HandleFunc("GET /api/movies", controllers.GetMovies(db))
	mux.HandleFunc("POST /api/movies", controllers.CreateMovie(db))
	mux.HandleFunc("PUT /api/movies/{id}", controllers.UpdateMovie(db))
	mux.HandleFunc("DELETE /api/movies/{id}", controllers.DeleteMovie(db))

	// WatchParties
	mux.HandleFunc("GET /api/parties", controllers.GetParties(db))
	mux.HandleFunc("POST /api/parties", controllers.CreateParty(db))
	mux.HandleFunc("GET /api/parties/{id}", controllers.GetPartyByID(db))
	mux.HandleFunc("PUT /api/parties/{id}", controllers.UpdateParty(db))
	mux.HandleFunc("DELETE /api/parties/{id}", controllers.DeleteParty(db))
	mux.HandleFunc("POST /api/parties/{id}/close", controllers.CloseParty(db))
	mux.HandleFunc("POST /api/parties/{id}/choose-movie", controllers.ChooseMovie(db))

	// Participants
	mux.HandleFunc("POST /api/parties/{id}/participants", controllers.CreateParticipant(db))
	mux.HandleFunc("GET /api/parties/{id}/participants", controllers.GetParticipants(db))

	// Invitations
	mux.HandleFunc("POST /api/parties/{id}/invitations", controllers.CreateInvitation(db))
	mux.HandleFunc("POST /api/invitations/{token}/accept", controllers.AcceptInvitation(db))

	// Commentaires
	mux.HandleFunc("POST /api/parties/{id}/comments", controllers.CreateComment(db))
	mux.HandleFunc("GET /api/parties/{id}/comments", controllers.GetComments(db))
	mux.HandleFunc("DELETE /api/comments/{id}", controllers.DeleteComment(db))

	// Swipes (cœur du projet)
	mux.HandleFunc("POST /api/parties/{id}/swipes", controllers.CreateSwipe(db))
	mux.HandleFunc("GET /api/parties/{id}/swipes", controllers.GetSwipes(db))

	// Recommandation (cœur du projet, cf. sujet) : calcul auto du film le plus liké
	mux.HandleFunc("POST /api/parties/{id}/recommendation/generate", controllers.GenerateRecommendation(db))
	mux.HandleFunc("GET /api/parties/{id}/recommendation", controllers.GetRecommendation(db))

	// Notation (bonus) : chaque participant note de 1 à 5 le film choisi/recommandé
	mux.HandleFunc("POST /api/parties/{id}/notations", controllers.CreateNotation(db))
	mux.HandleFunc("GET /api/parties/{id}/notations", controllers.GetNotations(db))

	return mux
}
