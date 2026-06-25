package controllers

import "net/http"

// Health permet de vérifier que le serveur répond.
// GET /api/health
func Health(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
