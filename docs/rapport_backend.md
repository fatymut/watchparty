# Rapport — Partie Backend (Amel)

> Brouillon à reformuler et intégrer dans le rapport global du groupe.
> Diagrammes associés : `docs/uml/class_diagram.puml`, `sequence_lancer_swipe.puml`, `sequence_swipe.puml`, `sequence_recommendation.puml`, `sequence_notation.puml`, `sequence_join_party.puml`, `object_diagram_recommendation.puml`.

## 1. Rôle du backend dans le projet

Le backend de WatchParty est une **API REST** qui expose les données et la logique métier de l'application. Il est consommé par le frontend React (swipe des films) et il est le seul à communiquer avec la base de données MySQL. Il assure trois responsabilités :

1. **Gérer les données** : utilisateurs, watch parties, films, participants, invitations, swipes, recommandations, notations, commentaires.
2. **Appliquer la logique métier** : gestion des participants/invitations, enregistrement des swipes, calcul automatique du film le plus aimé, et collecte des notes du film choisi.
3. **Sécuriser les échanges** : validation des entrées, hachage des mots de passe, authentification par token JWT, protection contre les injections SQL.

## 2. Choix techniques et justifications

| Choix | Justification |
|-------|---------------|
| **Go (langage)** | Langage compilé, performant et typé, adapté à une API. |
| **`net/http` natif (sans Gin)** | Depuis Go 1.22, `net/http` gère nativement le routage par méthode (`GET /api/...`) et les paramètres d'URL (`{id}` via `r.PathValue("id")`). Un framework comme Gin devient inutile : on garde une dépendance en moins et on comprend exactement ce que fait notre code. |
| **`database/sql` + driver `go-sql-driver/mysql`** | Bibliothèque standard pour le SQL, sans ORM. Cela nous oblige à écrire le SQL nous-mêmes, ce qui est formateur et donne un contrôle total sur les requêtes. |
| **MySQL** | Base relationnelle classique, parfaite pour des entités fortement reliées (clés étrangères entre users, parties, films, swipes). |
| **Docker Compose** | Permet à toute l'équipe de lancer la même base MySQL en une commande, sans installation locale (`docker compose up -d`). |
| **bcrypt** | Standard pour hacher les mots de passe ; on ne stocke jamais le mot de passe en clair. |

## 3. Architecture en couches

Le backend est organisé en couches, chacune ayant une responsabilité unique :

```
cmd/main.go              → point d'entrée : connexion DB, migration, seed, démarrage serveur
internal/router          → déclaration des routes (qui appelle quel controller)
internal/middlewares     → CORS (autorise le frontend à appeler l'API)
internal/controllers     → logique de chaque endpoint (validation + appel DB + réponse JSON)
internal/database        → connexion MySQL, création des tables, données de démo
internal/models          → structures de données (entités du domaine)
```

Cette séparation rend le code lisible et maintenable : pour ajouter une fonctionnalité, on sait exactement où intervenir.

## 4. Modèle de données

Neuf entités (voir le **diagramme de classes**) :

- **User** : un utilisateur (username, email, mot de passe haché).
- **WatchParty** : une soirée film créée par un utilisateur, avec un `chosenMovieId` optionnel désignant le film retenu pour la soirée (renseigné automatiquement par la recommandation, ou manuellement).
- **Movie** : un film pouvant être proposé.
- **Swipe** : le choix (`like` / `dislike`) d'un utilisateur sur un film, dans une party donnée. C'est la table centrale qui relie User, Movie et WatchParty.
- **Recommendation** : le film gagnant calculé automatiquement pour une party (cœur du projet, cf. sujet), conservé comme historique.
- **Notation** *(fonctionnalité bonus, non demandée par le sujet)* : la note (1 à 5) qu'un participant donne au film choisi/recommandé d'une party.
- **Participant** : relie un User à une WatchParty (relation many-to-many), avec un rôle (`creator` ou `member`). Le créateur d'une party y est automatiquement ajouté comme participant.
- **Invitation** : une invitation par email à rejoindre une party, identifiée par un token unique. Passe de `pending` à `accepted` une fois utilisée.
- **Comment** : un commentaire laissé par un User sur une WatchParty.

Les relations sont matérialisées par des **clés étrangères** en base, ce qui garantit la cohérence des données (on ne peut pas créer un swipe pour un film inexistant, etc.).

## 5. Fonctionnalité cœur : du swipe à la recommandation

### 5.1 Lancer la session de swipe (créateur uniquement, voir séquence « lancer_swipe »)

Avant de swiper, le créateur lance la session via `POST /api/parties/{id}/start-swipe` — route **protégée par JWT** (`middlewares.RequireAuth`). Le controller extrait l'id de l'utilisateur depuis le token (pas depuis le corps de la requête, contrairement au reste de l'API), le compare au `creatorId` de la party, et refuse avec `403` si ce n'est pas le créateur. La party passe alors en `status = 'active'`.

Chaque participant récupère ensuite son propre paquet de cartes via `GET /api/parties/{id}/movies?userId=`, qui exclut les films **déjà swipés par cet utilisateur précis** dans cette party (sous-requête `NOT IN` sur `swipes`). Deux participants voient donc des paquets différents selon leur progression individuelle — contrairement à `GET /api/movies`, qui reste la route générique (non filtrée) utilisée pour l'administration des films.

### 5.2 Enregistrer un swipe (voir séquence « swipe »)

Quand un participant swipe, le frontend envoie `POST /api/parties/{id}/swipes` avec `userId`, `movieId` et `value`. Le controller valide les données puis exécute :

```sql
INSERT INTO swipes (user_id, movie_id, watch_party_id, value)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE value = VALUES(value);
```

Grâce à la contrainte `UNIQUE (user_id, movie_id, watch_party_id)`, si l'utilisateur swipe deux fois le même film, son choix est **mis à jour** au lieu de créer un doublon.

### 5.3 Calculer le film le plus aimé (voir séquence « recommandation » et diagramme d'objets « object_diagram_recommendation »)

L'algorithme de recommandation tient en **une seule requête SQL** : on compte les `like` par film pour la party, et on garde celui qui en a le plus — exactement l'exemple du sujet (Interstellar / Inception / Titanic).

```sql
SELECT movie_id, COUNT(*) AS likes
FROM swipes
WHERE watch_party_id = ? AND value = 'like'
GROUP BY movie_id
ORDER BY likes DESC, movie_id ASC
LIMIT 1;
```

Le résultat est enregistré dans `recommendations` (historique), puis renvoyé au frontend qui affiche le film recommandé. Ce même appel met aussi à jour `watch_parties.chosen_movie_id`, pour que la notation bonus (5.4) sache quel film noter.

### 5.4 Noter le film choisi *(fonctionnalité bonus, voir séquence « notation »)*

En plus de la recommandation automatique demandée par le sujet, chaque participant peut donner une note manuelle de 1 à 5 au film choisi :

1. **Choisir le film** (repli manuel, si aucune recommandation n'a encore été générée) : le créateur désigne le film retenu via `POST /api/parties/{id}/choose-movie` `{movieId}` → `UPDATE watch_parties SET chosen_movie_id = ? WHERE id = ?`.
2. **Noter le film** : chaque participant envoie `POST /api/parties/{id}/notations` `{userId, rating}`. Le controller relit d'abord `chosen_movie_id` sur la party (erreur `400` si aucun film n'a encore été choisi/recommandé), puis enregistre :

```sql
INSERT INTO notations (watch_party_id, movie_id, user_id, rating)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE rating = VALUES(rating);
```

Grâce à la contrainte `UNIQUE (watch_party_id, user_id)`, un participant qui note deux fois **met à jour** sa note au lieu d'en créer une deuxième. `GET /api/parties/{id}/notations` renvoie la liste des notes ainsi que leur moyenne, calculée côté Go (pas en SQL) pour rester simple.

### 5.5 Rejoindre une party (participants et invitations, voir séquence « join_party »)

Deux façons de devenir participant d'une party :

1. **Automatique** : à la création de la party (`POST /api/parties`), le créateur est immédiatement inséré dans `participants` avec le rôle `creator`.
2. **Par invitation** : le créateur envoie une invitation (`POST /api/parties/{id}/invitations`) avec un email ; un token aléatoire est généré avec `crypto/rand` (imprévisible, contrairement à `math/rand`). L'invité utilise ce token pour rejoindre la party (`POST /api/invitations/{token}/accept`), ce qui crée sa ligne dans `participants` et marque l'invitation `accepted` pour empêcher sa réutilisation.

La contrainte `UNIQUE (watch_party_id, user_id)` sur `participants` empêche un utilisateur de rejoindre deux fois la même party.

### 5.6 Authentification par token (JWT)

`register` et `login` génèrent désormais un **token JWT** (HMAC-SHA256, `internal/auth/jwt.go`, valable 24h). Le middleware `middlewares.RequireAuth` vérifie le header `Authorization: Bearer <token>`, valide le token et injecte l'id de l'utilisateur dans le contexte de la requête. Une route protégée l'utilise pour l'instant : `GET /api/me`.

**Choix assumé et documenté** : les autres routes existantes (swipes, parties, comments...) ne sont pas verrouillées derrière ce token — elles continuent de recevoir `userId` explicitement dans le corps de la requête, comme avant. Protéger systématiquement toutes les routes aurait cassé l'intégration déjà fonctionnelle avec le frontend React (qui n'envoie pas encore le header `Authorization`), un risque jugé disproportionné à quelques jours de la soutenance. Le middleware est prêt et réutilisable (`middlewares.RequireAuth(handler)`) pour étendre la protection plus tard si le temps le permet.

## 6. Sécurité

- **Requêtes paramétrées** (`?`) partout : les valeurs ne sont jamais concaténées dans le SQL, ce qui empêche les **injections SQL**.
- **Mots de passe hachés** avec bcrypt : jamais stockés en clair, et jamais renvoyés dans les réponses JSON (tag `json:"-"`).
- **Validation des entrées** dans chaque controller (champs obligatoires, valeur de swipe limitée à `like`/`dislike`).
- **CORS** restreint à l'origine du frontend.
- **Token d'invitation** généré avec `crypto/rand` (aléatoire cryptographique) et invalidé après usage (`status = accepted`).
- **Notation bornée** : `rating` validé entre 1 et 5, et systématiquement rattachée au `chosen_movie_id` de la party (jamais un film au choix du client) grâce à une relecture serveur avant insertion.
- **JWT** signé HMAC-SHA256, expiration 24h, secret configurable via `JWT_SECRET`.
- **Autorisation par ownership** : `POST /api/parties/{id}/start-swipe` compare le `userId` du token JWT au `creatorId` de la party (`403` sinon) — première route où l'identité vient du token et non d'un champ envoyé par le client, donc non falsifiable.

## 7. Tests

L'API a été testée avec **Postman** (collection fournie dans `backend/postman/`). Scénario complet validé :
`health → register/login → création d'une party (créateur auto-participant) → ajout d'un participant → invitation puis acceptation → lancement de la session de swipe (créateur) → lecture des films restants par utilisateur → plusieurs swipes → génération de la recommandation (met aussi à jour le film choisi) → lecture de la recommandation → notation par plusieurs participants → lecture de la moyenne → ajout d'un commentaire`, ainsi que les cas d'erreur (mauvais mot de passe, email déjà utilisé, participant en doublon, invitation déjà utilisée, party sans like, notation sans film choisi, note hors de l'intervalle 1-5, lancement de session par un non-créateur).

## 8. Liste des endpoints

| Méthode | URL | Rôle |
|---------|-----|------|
| GET  | `/api/health` | Vérifier que l'API répond |
| POST | `/api/register` | Créer un compte (renvoie un token JWT) |
| POST | `/api/login` | Se connecter (renvoie un token JWT) |
| GET  | `/api/users` | Lister les utilisateurs |
| GET  | `/api/me` | 🔒 Utilisateur authentifié (route protégée par JWT) |
| GET / POST | `/api/movies` | Lister / ajouter des films |
| PUT / DELETE | `/api/movies/{id}` | Modifier / supprimer un film |
| GET / POST | `/api/parties` | Lister / créer des watch parties |
| GET  | `/api/parties/{id}` | Détail d'une party |
| PUT / DELETE | `/api/parties/{id}` | Modifier / supprimer une party |
| POST | `/api/parties/{id}/close` | Fermer une party (`status` → `closed`) |
| POST | `/api/parties/{id}/start-swipe` | 🔒 Lancer la session de swipe (créateur uniquement) |
| GET  | `/api/parties/{id}/movies?userId=` | Films restants à swiper pour cet utilisateur |
| GET / POST | `/api/parties/{id}/participants` | Lister / ajouter des participants |
| POST | `/api/parties/{id}/invitations` | Inviter un email à rejoindre une party |
| POST | `/api/invitations/{token}/accept` | Accepter une invitation |
| GET / POST | `/api/parties/{id}/comments` | Lister / ajouter des commentaires |
| DELETE | `/api/comments/{id}` | Supprimer un commentaire |
| POST / GET | `/api/parties/{id}/swipes` | Enregistrer / lister les swipes |
| POST | `/api/parties/{id}/recommendation/generate` | Calculer le film gagnant |
| GET  | `/api/parties/{id}/recommendation` | Récupérer la recommandation |
| POST | `/api/parties/{id}/choose-movie` | *(bonus)* Désigner manuellement le film choisi |
| GET / POST | `/api/parties/{id}/notations` | *(bonus)* Lister / donner une note (1-5) au film choisi |
