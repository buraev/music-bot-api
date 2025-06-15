package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"music-bot-api/internal/config"
	"net/http"
	"strings"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Link struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Tags []Tag  `json:"tags"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.Handle("/tags", withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var t Tag
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if t.Name == "" {
				http.Error(w, "Missing tag name", http.StatusBadRequest)
				return
			}
			t.ID = uuid.New().String()
			_, err := db.Exec("INSERT INTO tags (id, name) VALUES (?, ?)", t.ID, t.Name)
			if err != nil {
				log.Println("Insert tag error:", err)
				http.Error(w, "Insert error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(t)

		case http.MethodGet:
			rows, err := db.Query("SELECT id, name FROM tags ORDER BY name ASC")
			if err != nil {
				http.Error(w, "DB query error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var tags []Tag
			for rows.Next() {
				var t Tag
				if err := rows.Scan(&t.ID, &t.Name); err != nil {
					http.Error(w, "Scan error", http.StatusInternalServerError)
					return
				}
				tags = append(tags, t)
			}
			json.NewEncoder(w).Encode(tags)

		case http.MethodPut:
			var t Tag
			if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if t.ID == "" || t.Name == "" {
				http.Error(w, "Missing 'id' or 'name'", http.StatusBadRequest)
				return
			}

			var existingID string
			err := db.QueryRow("SELECT id FROM tags WHERE name = ? AND id != ?", t.Name, t.ID).Scan(&existingID)
			if err == nil {
				http.Error(w, "Tag name already exists", http.StatusConflict)
				return
			} else if err != sql.ErrNoRows {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}

			res, err := db.Exec("UPDATE tags SET name = ? WHERE id = ?", t.Name, t.ID)
			if err != nil {
				http.Error(w, "DB update error", http.StatusInternalServerError)
				return
			}
			count, _ := res.RowsAffected()
			if count == 0 {
				http.Error(w, "Tag not found", http.StatusNotFound)
				return
			}
			w.Write([]byte("Tag updated"))

		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "Missing tag ID", http.StatusBadRequest)
				return
			}
			_, err := db.Exec("DELETE FROM link_tags WHERE tag_id = ?", id)
			if err != nil {
				http.Error(w, "Delete link_tags error", http.StatusInternalServerError)
				return
			}
			res, err := db.Exec("DELETE FROM tags WHERE id = ?", id)
			if err != nil {
				http.Error(w, "Delete tag error", http.StatusInternalServerError)
				return
			}
			count, _ := res.RowsAffected()
			if count == 0 {
				http.Error(w, "Tag not found", http.StatusNotFound)
				return
			}
			w.Write([]byte("Tag deleted"))

		default:
			http.Error(w, "Only GET, POST, PUT, DELETE supported", http.StatusMethodNotAllowed)
		}
	})))

	http.Handle("/links", withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {

		case http.MethodPost:
			type IncomingTag struct {
				ID string `json:"id"`
			}

			type IncomingLink struct {
				URL  string        `json:"url"`
				Tags []IncomingTag `json:"tags"`
			}

			var incoming IncomingLink
			if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if incoming.URL == "" {
				http.Error(w, "Missing url", http.StatusBadRequest)
				return
			}

			var linkID string
			err := db.QueryRow("SELECT id FROM links WHERE url = ?", incoming.URL).Scan(&linkID)
			if err == sql.ErrNoRows {
				linkID = uuid.New().String()
				_, err = db.Exec("INSERT INTO links (id, url) VALUES (?, ?)", linkID, incoming.URL)
				if err != nil {
					log.Println("Insert link error:", err)
					http.Error(w, "Insert link error", http.StatusInternalServerError)
					return
				}
			} else if err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}

			var tagIDs []string
			for _, t := range incoming.Tags {
				if t.ID != "" {
					tagIDs = append(tagIDs, t.ID)
				}
			}

			if err := replaceLinkTags(db, linkID, tagIDs); err != nil {
				log.Println("Error updating tags:", err)
				http.Error(w, "Tag update error", http.StatusInternalServerError)
				return
			}

			tagRows, err := db.Query(`
				SELECT t.id, t.name
				FROM tags t
				JOIN link_tags lt ON t.id = lt.tag_id
				WHERE lt.link_id = ?
			`, linkID)
			if err != nil {
				log.Println("Failed to load tag info:", err)
				http.Error(w, "Failed to load tags", http.StatusInternalServerError)
				return
			}
			defer tagRows.Close()

			var fullTags []Tag
			for tagRows.Next() {
				var t Tag
				if err := tagRows.Scan(&t.ID, &t.Name); err != nil {
					http.Error(w, "Tag scan error", http.StatusInternalServerError)
					return
				}
				fullTags = append(fullTags, t)
			}

			result := Link{
				ID:   linkID,
				URL:  incoming.URL,
				Tags: fullTags,
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(result)

		case http.MethodGet:
			queryTags := r.URL.Query().Get("tags")
			var links []Link
			var err error
			if queryTags == "" {
				links, err = getAllLinksWithTags(db)
			} else {
				tags := strings.Split(queryTags, ",")
				links, err = getLinksByTags(db, tags)
			}
			if err != nil {
				http.Error(w, "DB query error", http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(links)

		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "Missing link ID", http.StatusBadRequest)
				return
			}

			// Удалить связи с тегами
			_, err := db.Exec("DELETE FROM link_tags WHERE link_id = ?", id)
			if err != nil {
				http.Error(w, "Failed to delete link_tags", http.StatusInternalServerError)
				return
			}

			// Удалить саму ссылку
			res, err := db.Exec("DELETE FROM links WHERE id = ?", id)
			if err != nil {
				http.Error(w, "Failed to delete link", http.StatusInternalServerError)
				return
			}
			count, _ := res.RowsAffected()
			if count == 0 {
				http.Error(w, "Link not found", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Link deleted"))

		default:
			http.Error(w, "Only GET, POST supported", http.StatusMethodNotAllowed)
		}
	})))

	log.Println("Server started on:", cfg.Port)
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

func replaceLinkTags(db *sql.DB, linkID string, tags []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM link_tags WHERE link_id = ?", linkID)
	if err != nil {
		return err
	}

	for _, tagID := range tags {
		_, err := tx.Exec("INSERT INTO link_tags (link_id, tag_id) VALUES (?, ?)", linkID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func getAllLinksWithTags(db *sql.DB) ([]Link, error) {
	rows, err := db.Query(`
		SELECT l.id, l.url, t.id, t.name
		FROM links l
		LEFT JOIN link_tags lt ON l.id = lt.link_id
		LEFT JOIN tags t ON lt.tag_id = t.id
		ORDER BY l.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	linkMap := make(map[string]*Link)

	for rows.Next() {
		var linkID, url sql.NullString
		var tagID, tagName sql.NullString

		if err := rows.Scan(&linkID, &url, &tagID, &tagName); err != nil {
			return nil, err
		}

		if !linkID.Valid || !url.Valid {
			continue
		}

		l, exists := linkMap[linkID.String]
		if !exists {
			l = &Link{
				ID:   linkID.String,
				URL:  url.String,
				Tags: []Tag{},
			}
			linkMap[linkID.String] = l
		}

		if tagID.Valid && tagName.Valid {
			l.Tags = append(l.Tags, Tag{
				ID:   tagID.String,
				Name: tagName.String,
			})
		}
	}

	var links []Link
	for _, l := range linkMap {
		links = append(links, *l)
	}
	return links, nil
}

func getLinksByTags(db *sql.DB, tagNames []string) ([]Link, error) {
	if len(tagNames) == 0 {
		return []Link{}, nil
	}

	placeholders := strings.Repeat("?,", len(tagNames))
	placeholders = placeholders[:len(placeholders)-1]

	query := `
		SELECT l.id, l.url, t.id, t.name
		FROM links l
		JOIN link_tags lt ON l.id = lt.link_id
		JOIN tags t ON lt.tag_id = t.id
		WHERE t.name IN (` + placeholders + `)
	`

	args := make([]interface{}, len(tagNames))
	for i, tag := range tagNames {
		args[i] = tag
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	linkMap := make(map[string]*Link)

	for rows.Next() {
		var linkID, url, tagID, tagName string
		if err := rows.Scan(&linkID, &url, &tagID, &tagName); err != nil {
			return nil, err
		}

		link, exists := linkMap[linkID]
		if !exists {
			link = &Link{
				ID:   linkID,
				URL:  url,
				Tags: []Tag{},
			}
			linkMap[linkID] = link
		}

		link.Tags = append(link.Tags, Tag{
			ID:   tagID,
			Name: tagName,
		})
	}

	var result []Link
	for _, link := range linkMap {
		tagNameSet := make(map[string]bool)
		for _, t := range link.Tags {
			tagNameSet[t.Name] = true
		}
		matchCount := 0
		for _, wanted := range tagNames {
			if tagNameSet[wanted] {
				matchCount++
			}
		}
		if matchCount == len(tagNames) {
			result = append(result, *link)
		}
	}

	return result, nil
}
