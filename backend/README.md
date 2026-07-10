# WatchParty — Backend

API REST pour l'application WatchParty (organiser une soirée film, swiper les films, recommander le préféré du groupe).

## Stack

- **Go** (classique, sans framework) — routing natif `net/http` (Go 1.22+)
- **`database/sql`** + driver `github.com/go-sql-driver/mysql`
- **MySQL 8** via Docker Compose
- JSON via `encoding/json`

> Choix assumé : **pas de Gin, pas de Gorm, pas de PostgreSQL.** Le routing par méthode (`GET /api/...`) et les paramètres d'URL (`{id}` → `r.PathValue("id")`) sont fournis nativement par `net/http` depuis Go 1.22.

## Prérequis

- Go 1.22+ (testé avec 1.25)
- Docker + Docker Compose

## Lancer le projet

```bash
# 1. Démarrer MySQL (depuis la racine du repo)
docker compose up -d

# 2. Lancer le serveur (depuis backend/) sur le port 8082
# PowerShell :
$env:PORT="8082"; go run ./cmd
# Bash / Git Bash :
PORT=8082 go run ./cmd
```

Le serveur démarre sur http://localhost:8082. Les tables sont créées automatiquement au démarrage (`database.Migrate`), et des données de démo (6 films + un utilisateur `demo@watchparty.com` / `password123`) sont insérées si les tables sont vides (`database.Seed`).

> **Port :** le port est configurable via la variable d'environnement `PORT` (défaut `8080`). On utilise **8082** sur ce projet car le port 8080 est déjà occupé par le listener Oracle (TNSLSNR) sur la machine de dev.

## Configuration

Connexion définie dans `internal/database/database.go` (alignée sur `docker-compose.yml`) :

| Paramètre | Valeur |
|-----------|--------|
| Host      | localhost |
| Port      | 3306 |
| User      | watchparty_user |
| Password  | watchparty_password |
| Database  | watchparty_db |

## Architecture

```
cmd/main.go                  point d'entrée : connexion DB, migration, serveur
internal/
  database/database.go       connexion MySQL + création des tables
  router/router.go           déclaration des routes (net/http natif)
  middlewares/cors.go        middleware CORS (autorise le front Vite :5173)
  controllers/               logique des endpoints
    response.go              helpers WriteJSON / WriteError
    health_controller.go
    movie_controller.go
    party_controller.go
    swipe_controller.go            <- enregistrement des swipes
    recommendation_controller.go   <- calcul du film le plus liké
    participant_controller.go      <- gestion des participants d'une party
    invitation_controller.go       <- invitations par email + acceptation
    comment_controller.go          <- commentaires sur une party
  models/                    structs Go (User, Movie, WatchParty, Swipe, Recommendation,
                              Participant, Invitation, Comment)
```

## Endpoints

| Méthode | URL | Description |
|---------|-----|-------------|
| GET  | `/api/health` | Test de disponibilité |
| POST | `/api/register` | Créer un compte (mot de passe hashé bcrypt) |
| POST | `/api/login` | Se connecter |
| GET  | `/api/users` | Liste des utilisateurs |
| GET  | `/api/movies` | Liste des films |
| POST | `/api/movies` | Ajouter un film |
| GET  | `/api/parties` | Liste des watch parties |
| POST | `/api/parties` | Créer une watch party |
| GET  | `/api/parties/{id}` | Détail d'une watch party |
| POST | `/api/parties/{id}/swipes` | Enregistrer un swipe (`like`/`dislike`) |
| GET  | `/api/parties/{id}/swipes` | Lister les swipes d'une party |
| POST | `/api/parties/{id}/recommendation/generate` | Calculer le film le plus liké |
| GET  | `/api/parties/{id}/recommendation` | Dernière recommandation (avec le film) |
| POST | `/api/parties/{id}/participants` | Ajouter un participant à une party |
| GET  | `/api/parties/{id}/participants` | Lister les participants d'une party |
| POST | `/api/parties/{id}/invitations` | Inviter un email à rejoindre une party (génère un token) |
| POST | `/api/invitations/{token}/accept` | Accepter une invitation → crée le participant |
| POST | `/api/parties/{id}/comments` | Ajouter un commentaire sur une party |
| GET  | `/api/parties/{id}/comments` | Lister les commentaires d'une party |

### Exemples de corps de requête

**POST /api/movies**
```json
{ "title": "Inception", "genre": "Sci-Fi", "duration": 148, "releaseYear": 2010, "synopsis": "...", "posterUrl": "https://..." }
```

**POST /api/parties**
```json
{ "title": "Soirée du vendredi", "description": "Film d'horreur", "creatorId": 1 }
```

**POST /api/parties/{id}/swipes**
```json
{ "userId": 1, "movieId": 2, "value": "like" }
```

**POST /api/parties/{id}/participants**
```json
{ "userId": 3 }
```

**POST /api/parties/{id}/invitations**
```json
{ "email": "ami@example.com" }
```

**POST /api/invitations/{token}/accept**
```json
{ "userId": 3 }
```

**POST /api/parties/{id}/comments**
```json
{ "userId": 1, "content": "Hâte de voir ce film !" }
```

## Algorithme de recommandation

Une seule requête SQL : on compte les `like` par film pour la party, on garde le meilleur.

```sql
SELECT movie_id, COUNT(*) AS likes
FROM swipes
WHERE watch_party_id = ? AND value = 'like'
GROUP BY movie_id
ORDER BY likes DESC, movie_id ASC
LIMIT 1;
```

## Sécurité

- Requêtes **paramétrées** (`?`) partout → protection contre les injections SQL.
- Contrainte `UNIQUE (user_id, movie_id, watch_party_id)` sur `swipes` → un seul swipe par user/film/party (re-swipe = mise à jour via `ON DUPLICATE KEY UPDATE`).
- Contrainte `UNIQUE (watch_party_id, user_id)` sur `participants` → un utilisateur ne peut pas rejoindre deux fois la même party.
- Token d'invitation généré avec `crypto/rand` (aléatoire cryptographique, pas `math/rand`) et marqué `accepted` après usage pour empêcher la réutilisation.
