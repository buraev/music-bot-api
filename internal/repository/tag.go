package repository

import (
	"database/sql"
	"errors"
	"log"
	"music-bot-api/internal/model"
)

type TagRepo struct {
	db *sql.DB
}

func NewTagRepo(db *sql.DB) *TagRepo {
	return &TagRepo{db: db}
}

func (r *TagRepo) CreateTag(t *model.Tag) error {
	_, err := r.db.Exec("INSERT INTO tags (id, name) VALUES (?, ?)", t.ID, t.Name)
	return err
}

func (r *TagRepo) GetAllTags() ([]model.Tag, error) {
	rows, err := r.db.Query("SELECT id, name FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var t model.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (r *TagRepo) UpdateTag(t *model.Tag) error {
	// Проверка уникальности имени
	var existingID string
	err := r.db.QueryRow("SELECT id FROM tags WHERE name = ? AND id != ?", t.Name, t.ID).Scan(&existingID)
	if err == nil {
		return errors.New("tag name already exists")
	} else if err != sql.ErrNoRows {
		return err
	}

	res, err := r.db.Exec("UPDATE tags SET name = ? WHERE id = ?", t.Name, t.ID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("tag not found")
	}
	return nil
}

func (r *TagRepo) DeleteTag(id string) error {
	_, err := r.db.Exec("DELETE FROM link_tags WHERE tag_id = ?", id)
	if err != nil {
		log.Println("Failed deleting link_tags:", err)
		return err
	}
	res, err := r.db.Exec("DELETE FROM tags WHERE id = ?", id)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("tag not found")
	}
	return nil
}
