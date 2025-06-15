package service

import (
	"errors"
	"music-bot-api/internal/model"
	"music-bot-api/internal/repository"

	"github.com/google/uuid"
)

type TagService struct {
	repo *repository.TagRepo
}

func NewTagService(repo *repository.TagRepo) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) CreateTag(name string) (*model.Tag, error) {
	if name == "" {
		return nil, errors.New("tag name is required")
	}
	t := &model.Tag{
		ID:   uuid.New().String(),
		Name: name,
	}
	if err := s.repo.CreateTag(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TagService) GetAllTags() ([]model.Tag, error) {
	return s.repo.GetAllTags()
}

func (s *TagService) UpdateTag(t *model.Tag) error {
	if t.ID == "" || t.Name == "" {
		return errors.New("tag id and name are required")
	}
	return s.repo.UpdateTag(t)
}

func (s *TagService) DeleteTag(id string) error {
	if id == "" {
		return errors.New("tag id is required")
	}
	return s.repo.DeleteTag(id)
}
