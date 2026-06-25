# Rapport — Partie Backend (Amel)

> Brouillon à reformuler et intégrer dans le rapport global du groupe.
> Diagrammes associés : `docs/uml/class_diagram.puml`, `sequence_swipe.puml`, `sequence_recommendation.puml`.

## 1. Rôle du backend dans le projet

Le backend de WatchParty est une **API REST** qui expose les données et la logique métier de l'application. Il est consommé par le frontend React (swipe des films) et il est le seul à communiquer avec la base de données MySQL. Il assure trois responsabilités :

1. **Gérer les données** : utilisateurs, watch parties, films, swipes, recommandations.
2. **Appliquer la logique métier** : enregistrement des swipes et calcul du film le plus aimé.
3. **Sécuriser les échanges** : validation des entrées, hachage des mots de passe, protection contre les injections SQL.

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

Cinq entités principales (voir le **diagramme de classes**) :

- **User** : un utilisateur (username, email, mot de passe haché).
- **WatchParty** : une soirée film créée par un utilisateur.
- **Movie** : un film pouvant être proposé.
- **Swipe** : le choix (`like` / `dislike`) d'un utilisateur sur un film, dans une party donnée. C'est la table centrale qui relie User, Movie et WatchParty.
- **Recommendation** : le film gagnant calculé pour une party, conservé comme historique.

Les relations sont matérialisées par des **clés étrangères** en base, ce qui garantit la cohérence des données (on ne peut pas créer un swipe pour un film inexistant, etc.).

## 5. Fonctionnalité cœur : du swipe à la recommandation

### 5.1 Enregistrer un swipe (voir séquence « swipe »)

Quand un participant swipe, le frontend envoie `POST /api/parties/{id}/swipes` avec `userId`, `movieId` et `value`. Le controller valide les données puis exécute :

```sql
INSERT INTO swipes (user_id, movie_id, watch_party_id, value)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE value = VALUES(value);
```

Grâce à la contrainte `UNIQUE (user_id, movie_id, watch_party_id)`, si l'utilisateur swipe deux fois le même film, son choix est **mis à jour** au lieu de créer un doublon.

### 5.2 Calculer le film le plus aimé (voir séquence « recommandation »)

L'algorithme de recommandation tient en **une seule requête SQL** : on compte les `like` par film pour la party, et on garde celui qui en a le plus.

```sql
SELECT movie_id, COUNT(*) AS likes
FROM swipes
WHERE watch_party_id = ? AND value = 'like'
GROUP BY movie_id
ORDER BY likes DESC, movie_id ASC
LIMIT 1;
```

Le résultat est ensuite enregistré dans la table `recommendations`, puis renvoyé au frontend qui affiche le film recommandé au groupe.

## 6. Sécurité

- **Requêtes paramétrées** (`?`) partout : les valeurs ne sont jamais concaténées dans le SQL, ce qui empêche les **injections SQL**.
- **Mots de passe hachés** avec bcrypt : jamais stockés en clair, et jamais renvoyés dans les réponses JSON (tag `json:"-"`).
- **Validation des entrées** dans chaque controller (champs obligatoires, valeur de swipe limitée à `like`/`dislike`).
- **CORS** restreint à l'origine du frontend.

## 7. Tests

L'API a été testée avec **Postman** (collection fournie dans `backend/postman/`). Scénario complet validé :
`health → register/login → création d'une party → plusieurs swipes → génération de la recommandation → lecture du film recommandé`, ainsi que les cas d'erreur (mauvais mot de passe, email déjà utilisé, party sans like).

## 8. Liste des endpoints

| Méthode | URL | Rôle |
|---------|-----|------|
| GET  | `/api/health` | Vérifier que l'API répond |
| POST | `/api/register` | Créer un compte |
| POST | `/api/login` | Se connecter |
| GET  | `/api/users` | Lister les utilisateurs |
| GET / POST | `/api/movies` | Lister / ajouter des films |
| GET / POST | `/api/parties` | Lister / créer des watch parties |
| GET  | `/api/parties/{id}` | Détail d'une party |
| POST / GET | `/api/parties/{id}/swipes` | Enregistrer / lister les swipes |
| POST | `/api/parties/{id}/recommendation/generate` | Calculer le film gagnant |
| GET  | `/api/parties/{id}/recommendation` | Récupérer la recommandation |
