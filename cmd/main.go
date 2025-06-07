package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"music-bot-api/internal/config"
	"net/http"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Genre struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		log.Fatal("Ошибка открытия БД:", err)
	}
	defer db.Close()

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS genres (
            id TEXT PRIMARY KEY,
            name TEXT UNIQUE,
            url TEXT
        )
    `)
	if err != nil {
		log.Fatal("Ошибка создания таблицы:", err)
	}

	http.Handle("/genres", withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {

		case http.MethodPost:
			var genres []Genre
			if err := json.NewDecoder(r.Body).Decode(&genres); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			tx, err := db.Begin()
			if err != nil {
				http.Error(w, "Transaction error", http.StatusInternalServerError)
				return
			}

			stmt, err := tx.Prepare(`
                INSERT INTO genres (id, name, url)
                VALUES (?, ?, ?)
                ON CONFLICT(name) DO UPDATE SET url = excluded.url
            `)
			if err != nil {
				tx.Rollback()
				http.Error(w, "Prepare error", http.StatusInternalServerError)
				return
			}
			defer stmt.Close()

			for _, g := range genres {
				id := uuid.New().String()
				_, err := stmt.Exec(id, g.Name, g.URL)
				if err != nil {
					tx.Rollback()
					http.Error(w, "Insert error", http.StatusInternalServerError)
					return
				}
			}

			tx.Commit()
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("Жанры добавлены или обновлены"))

		case http.MethodGet:
			rows, err := db.Query("SELECT id, name, url FROM genres ORDER BY name ASC")
			if err != nil {
				http.Error(w, "DB query error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var genres []Genre
			for rows.Next() {
				var g Genre
				if err := rows.Scan(&g.ID, &g.Name, &g.URL); err != nil {
					http.Error(w, "Scan error", http.StatusInternalServerError)
					return
				}
				genres = append(genres, g)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(genres)

		case http.MethodPut:
			var g Genre
			if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if g.ID == "" || g.Name == "" || g.URL == "" {
				http.Error(w, "Missing 'id', 'name' or 'url'", http.StatusBadRequest)
				return
			}

			// Проверка конфликта: существует ли другой жанр с таким именем?
			var existingID string
			err := db.QueryRow("SELECT id FROM genres WHERE name = ? AND id != ?", g.Name, g.ID).Scan(&existingID)
			if err == nil {
				http.Error(w, "Genre name already exists", http.StatusConflict)
				return
			} else if err != sql.ErrNoRows {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}

			// Обновление
			res, err := db.Exec("UPDATE genres SET name = ?, url = ? WHERE id = ?", g.Name, g.URL, g.ID)
			if err != nil {
				http.Error(w, "DB update error", http.StatusInternalServerError)
				return
			}

			count, _ := res.RowsAffected()
			if count == 0 {
				http.Error(w, "Genre not found", http.StatusNotFound)
				return
			}

			w.Write([]byte("Жанр обновлён по ID: " + g.ID))

		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "Missing genre ID", http.StatusBadRequest)
				return
			}

			res, err := db.Exec("DELETE FROM genres WHERE id = ?", id)
			if err != nil {
				http.Error(w, "Delete error", http.StatusInternalServerError)
				return
			}

			count, _ := res.RowsAffected()
			if count == 0 {
				http.Error(w, "Genre not found", http.StatusNotFound)
				return
			}

			w.Write([]byte("Жанр удалён (ID): " + id))

		default:
			http.Error(w, "Only GET, POST, PUT, DELETE supported", http.StatusMethodNotAllowed)
		}
	})))

	log.Println("Сервер запущен на http://localhost" + cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
