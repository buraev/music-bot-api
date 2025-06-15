package repository

import (
	"database/sql"
	"music-bot-api/internal/model"
	"strings"

	"github.com/google/uuid"
)

type LinkRepo struct {
	db *sql.DB
}

func NewLinkRepo(db *sql.DB) *LinkRepo {
	return &LinkRepo{db: db}
}

// Insert or get existing link by URL
func (r *LinkRepo) GetOrCreateLink(url string) (string, error) {
	var linkID string
	err := r.db.QueryRow("SELECT id FROM links WHERE url = ?", url).Scan(&linkID)
	if err == sql.ErrNoRows {
		linkID = uuid.New().String()
		_, err = r.db.Exec("INSERT INTO links (id, url) VALUES (?, ?)", linkID, url)
		if err != nil {
			return "", err
		}
		return linkID, nil
	} else if err != nil {
		return "", err
	}
	return linkID, nil
}

func (r *LinkRepo) ReplaceLinkTags(linkID string, tagIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM link_tags WHERE link_id = ?", linkID)
	if err != nil {
		return err
	}

	for _, tagID := range tagIDs {
		_, err := tx.Exec("INSERT INTO link_tags (link_id, tag_id) VALUES (?, ?)", linkID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *LinkRepo) GetAllLinksWithTags() ([]model.Link, error) {
	rows, err := r.db.Query(`
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

	linkMap := make(map[string]*model.Link)
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
			l = &model.Link{
				ID:   linkID.String,
				URL:  url.String,
				Tags: []model.Tag{},
			}
			linkMap[linkID.String] = l
		}

		if tagID.Valid && tagName.Valid {
			l.Tags = append(l.Tags, model.Tag{
				ID:   tagID.String,
				Name: tagName.String,
			})
		}
	}

	links := make([]model.Link, 0, len(linkMap))
	for _, link := range linkMap {
		links = append(links, *link)
	}
	return links, nil
}

func (r *LinkRepo) GetLinksByTags(tagNames []string) ([]model.Link, error) {
	if len(tagNames) == 0 {
		return []model.Link{}, nil
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

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	linkMap := make(map[string]*model.Link)
	for rows.Next() {
		var linkID, url, tagID, tagName string
		if err := rows.Scan(&linkID, &url, &tagID, &tagName); err != nil {
			return nil, err
		}

		link, exists := linkMap[linkID]
		if !exists {
			link = &model.Link{
				ID:   linkID,
				URL:  url,
				Tags: []model.Tag{},
			}
			linkMap[linkID] = link
		}

		link.Tags = append(link.Tags, model.Tag{
			ID:   tagID,
			Name: tagName,
		})
	}

	var result []model.Link
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

func (r *LinkRepo) DeleteLink(id string) error {
	_, err := r.db.Exec("DELETE FROM link_tags WHERE link_id = ?", id)
	if err != nil {
		return err
	}

	res, err := r.db.Exec("DELETE FROM links WHERE id = ?", id)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}
