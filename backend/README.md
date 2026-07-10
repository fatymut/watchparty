# WatchParty — Backend

API REST pour l'application WatchParty (organiser une soirée film, swiper les films, recommander le préféré du groupe, et le noter en bonus).

## Stack

- **Go** (classique, sans framework) — routing natif `net/http` (Go 1.22+)
- **`database/sql`** + driver `github.com/go-sql-driver/mysql`
- **MySQL 8** via Docker Compose
- JSON via `encoding/json`

> Choix assumé : **pas de Gin, pas de Gorm, pas de PostgreSQL.** Le routing par méthode (`GET /api/...`) et les paramètres d'URL (`{id}` → `r.PathValue("id")`) sont fournis nativement par `net/http` depuis Go 1.22.

## CI

`.github/workflows/backend.yml` lance `go build`, `go vet` et `go test` à chaque push/PR touchant `backend/`.

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

Variable d'environnement optionnelle : `JWT_SECRET` (secret de signature des tokens). Une valeur par défaut est utilisée en développement si elle n'est pas définie — **à changer en production**.

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
    recommendation_controller.go   <- calcul du film le plus liké (cœur du projet)
    notation_controller.go         <- notation (1-5) du film choisi (bonus)
    participant_controller.go      <- gestion des participants d'une party
    invitation_controller.go       <- invitations par email + acceptation
    comment_controller.go          <- commentaires sur une party
  models/                    structs Go (User, Movie, WatchParty, Swipe, Recommendation,
                              Notation, Participant, Invitation, Comment)
```

## Endpoints

| Méthode | URL | Description |
|---------|-----|-------------|
| GET  | `/api/health` | Test de disponibilité |
| POST | `/api/register` | Créer un compte (mot de passe hashé bcrypt, renvoie un token JWT) |
| POST | `/api/login` | Se connecter (renvoie un token JWT) |
| GET  | `/api/users` | Liste des utilisateurs |
| GET  | `/api/me` | 🔒 Utilisateur authentifié (nécessite `Authorization: Bearer <token>`) |
| GET  | `/api/movies` | Liste des films |
| POST | `/api/movies` | Ajouter un film |
| PUT  | `/api/movies/{id}` | Modifier un film |
| DELETE | `/api/movies/{id}` | Supprimer un film |
| GET  | `/api/parties` | Liste des watch parties |
| POST | `/api/parties` | Créer une watch party |
| GET  | `/api/parties/{id}` | Détail d'une watch party |
| PUT  | `/api/parties/{id}` | Modifier une watch party |
| DELETE | `/api/parties/{id}` | Supprimer une watch party (cascade) |
| POST | `/api/parties/{id}/close` | Fermer une watch party (`status` → `closed`) |
| POST | `/api/parties/{id}/swipes` | Enregistrer un swipe (`like`/`dislike`) |
| GET  | `/api/parties/{id}/swipes` | Lister les swipes d'une party |
| POST | `/api/parties/{id}/recommendation/generate` | Calculer le film le plus liké |
| GET  | `/api/parties/{id}/recommendation` | Dernière recommandation (avec le film) |
| POST | `/api/parties/{id}/choose-movie` | *(bonus)* Désigner manuellement le film choisi |
| POST | `/api/parties/{id}/notations` | *(bonus)* Noter (1 à 5) le film choisi |
| GET  | `/api/parties/{id}/notations` | *(bonus)* Lister les notes + moyenne du film choisi |
| POST | `/api/parties/{id}/participants` | Ajouter un participant à une party |
| GET  | `/api/parties/{id}/participants` | Lister les participants d'une party |
| POST | `/api/parties/{id}/invitations` | Inviter un email à rejoindre une party (génère un token) |
| POST | `/api/invitations/{token}/accept` | Accepter une invitation → crée le participant |
| POST | `/api/parties/{id}/comments` | Ajouter un commentaire sur une party |
| GET  | `/api/parties/{id}/comments` | Lister les commentaires d'une party |
| DELETE | `/api/comments/{id}` | Supprimer un commentaire |

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

**POST /api/parties/{id}/choose-movie**
```json
{ "movieId": 2 }
```

**POST /api/parties/{id}/notations**
```json
{ "userId": 1, "rating": 5 }
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

Le film gagnant devient aussi `watch_parties.chosen_movie_id`, ce qui alimente la notation bonus ci-dessous.

## Notation (bonus)

En plus de la recommandation automatique, chaque participant peut donner une **note manuelle de 1 à 5** au film choisi/recommandé (table `notations`). La moyenne est calculée côté Go à la lecture (`GET /api/parties/{id}/notations`). Si aucune recommandation n'a encore été générée, le créateur peut désigner le film manuellement via `POST /api/parties/{id}/choose-movie`.

## Authentification (JWT)

`register` et `login` renvoient un **token JWT** (`internal/auth/jwt.go`, HMAC-SHA256, valable 24h, secret configurable via `JWT_SECRET`). Pour appeler une route protégée, l'ajouter en header :

```
Authorization: Bearer <token>
```

Pour l'instant, une seule route est protégée : `GET /api/me` (via `middlewares.RequireAuth`). **Choix assumé** : les autres routes (`swipes`, `parties`, `comments`, etc.) ne sont pas verrouillées derrière le token, elles continuent de recevoir `userId` explicitement dans le corps de la requête. Verrouiller toutes les routes aurait cassé l'intégration déjà fonctionnelle avec le frontend (qui n'envoie pas encore le header `Authorization`) à quelques jours de la soutenance. `RequireAuth` est réutilisable pour protéger d'autres routes plus tard si besoin (`middlewares.RequireAuth(monHandler)`).

## Sécurité

- Requêtes **paramétrées** (`?`) partout → protection contre les injections SQL.
- Contrainte `UNIQUE (user_id, movie_id, watch_party_id)` sur `swipes` → un seul swipe par user/film/party (re-swipe = mise à jour via `ON DUPLICATE KEY UPDATE`).
- Contrainte `UNIQUE (watch_party_id, user_id)` sur `participants` → un utilisateur ne peut pas rejoindre deux fois la même party.
- Token d'invitation généré avec `crypto/rand` (aléatoire cryptographique, pas `math/rand`) et marqué `accepted` après usage pour empêcher la réutilisation.
- Contrainte `UNIQUE (watch_party_id, user_id)` sur `notations` → un seul vote par participant/party (re-noter met à jour la note). `rating` validé entre 1 et 5, et toujours associé au `chosen_movie_id` de la party (pas de film arbitraire côté client).
