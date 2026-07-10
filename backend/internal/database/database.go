package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

func Connect() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		"watchparty_user",
		"watchparty_password",
		"localhost",
		"3306",
		"watchparty_db",
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Connexion MySQL réussie")
	return db, nil
}

func Migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(100) NOT NULL,
		email VARCHAR(150) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS movies (
		id INT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(150) NOT NULL,
		genre VARCHAR(100),
		duration INT,
		release_year INT,
		synopsis TEXT,
		poster_url TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS watch_parties (
		id INT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(150) NOT NULL,
		description TEXT,
		date DATETIME NULL,
		status VARCHAR(50) DEFAULT 'draft',
		creator_id INT NULL,
		chosen_movie_id INT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE SET NULL,
		FOREIGN KEY (chosen_movie_id) REFERENCES movies(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS swipes (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_id INT NOT NULL,
		movie_id INT NOT NULL,
		watch_party_id INT NOT NULL,
		value VARCHAR(20) NOT NULL,
		swiped_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE,
		FOREIGN KEY (watch_party_id) REFERENCES watch_parties(id) ON DELETE CASCADE,
		UNIQUE KEY unique_swipe (user_id, movie_id, watch_party_id)
	);

	CREATE TABLE IF NOT EXISTS recommendations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		watch_party_id INT NOT NULL,
		movie_id INT NOT NULL,
		likes_count INT NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (watch_party_id) REFERENCES watch_parties(id) ON DELETE CASCADE,
		FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS notations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		watch_party_id INT NOT NULL,
		movie_id INT NOT NULL,
		user_id INT NOT NULL,
		rating INT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (watch_party_id) REFERENCES watch_parties(id) ON DELETE CASCADE,
		FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE KEY unique_notation (watch_party_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS participants (
		id INT AUTO_INCREMENT PRIMARY KEY,
		watch_party_id INT NOT NULL,
		user_id INT NOT NULL,
		role VARCHAR(20) DEFAULT 'member',
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (watch_party_id) REFERENCES watch_parties(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE KEY unique_participant (watch_party_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS invitations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		watch_party_id INT NOT NULL,
		invited_email VARCHAR(150) NOT NULL,
		token VARCHAR(100) NOT NULL UNIQUE,
		status VARCHAR(20) DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (watch_party_id) REFERENCES watch_parties(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INT AUTO_INCREMENT PRIMARY KEY,
		watch_party_id INT NOT NULL,
		user_id INT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (watch_party_id) REFERENCES watch_parties(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	// Migration légère pour les bases déjà existantes (créées avant l'ajout de chosen_movie_id).
	// CREATE TABLE IF NOT EXISTS ne modifie pas une table déjà existante ; MySQL ne supporte pas
	// "ADD COLUMN IF NOT EXISTS" (contrairement à MariaDB), donc on vérifie nous-mêmes via
	// information_schema avant d'altérer.
	var columnExists int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_schema = DATABASE() AND table_name = 'watch_parties' AND column_name = 'chosen_movie_id'`,
	).Scan(&columnExists)
	if err != nil {
		return err
	}
	if columnExists == 0 {
		if _, err := db.Exec(`ALTER TABLE watch_parties ADD COLUMN chosen_movie_id INT NULL`); err != nil {
			return err
		}
		if _, err := db.Exec(
			`ALTER TABLE watch_parties ADD CONSTRAINT fk_watch_parties_chosen_movie
			 FOREIGN KEY (chosen_movie_id) REFERENCES movies(id) ON DELETE SET NULL`,
		); err != nil {
			return err
		}
	}

	log.Println("Tables créées ou déjà existantes")
	return nil
}

// Seed insère des données de démonstration (films + un utilisateur de test)
// uniquement si les tables correspondantes sont vides. Idempotent.
func Seed(db *sql.DB) error {
	// --- Utilisateur de démo ---
	var userCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := db.Exec(
			`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
			"demo", "demo@watchparty.com", string(hash),
		); err != nil {
			return err
		}
		log.Println("Seed : utilisateur de démo créé (demo@watchparty.com / password123)")
	}

	// --- Films de démo ---
	var movieCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM movies`).Scan(&movieCount); err != nil {
		return err
	}
	if movieCount == 0 {
		movies := []struct {
			title       string
			genre       string
			duration    int
			releaseYear int
			synopsis    string
			posterURL   string
		}{
			{"Inception", "Sci-Fi", 148, 2010, "Un voleur s'infiltre dans les rêves.", "https://image.tmdb.org/t/p/w500/9gk7adHYeDvHkCSEqAvQNLV5Uge.jpg"},
			{"The Matrix", "Sci-Fi", 136, 1999, "Un hacker découvre la vraie nature de la réalité.", "https://image.tmdb.org/t/p/w500/f89U3ADr1oiB1s9GkdPOEpXUk5H.jpg"},
			{"Parasite", "Thriller", 132, 2019, "Une famille pauvre s'immisce chez des riches.", "https://image.tmdb.org/t/p/w500/7IiTTgloJzvGI1TAYymCfbfl3vT.jpg"},
			{"Interstellar", "Sci-Fi", 169, 2014, "Un voyage à travers un trou de ver pour sauver l'humanité.", "https://image.tmdb.org/t/p/w500/gEU2QniE6E77NI6lCU6MxlNBvIx.jpg"},
			{"The Dark Knight", "Action", 152, 2008, "Batman affronte le Joker.", "https://image.tmdb.org/t/p/w500/qJ2tW6WMUDux911r6m7haRef0WH.jpg"},
			{"Spirited Away", "Animation", 125, 2001, "Une fillette piégée dans un monde d'esprits.", "https://image.tmdb.org/t/p/w500/39wmItIWsg5sZMyRUHLkWBcuVCM.jpg"},
		}
		for _, m := range movies {
			if _, err := db.Exec(
				`INSERT INTO movies (title, genre, duration, release_year, synopsis, poster_url)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				m.title, m.genre, m.duration, m.releaseYear, m.synopsis, m.posterURL,
			); err != nil {
				return err
			}
		}
		log.Printf("Seed : %d films de démo insérés", len(movies))
	}

	return nil
}