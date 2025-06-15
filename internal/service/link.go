package service

import (
	"errors"
	"music-bot-api/internal/model"
	"music-bot-api/internal/repository"
)

type LinkService struct {
	repo *repository.LinkRepo
}

func NewLinkService(repo *repository.LinkRepo) *LinkService {
	return &LinkService{repo: repo}
}

func (s *LinkService) CreateOrUpdateLink(url string, tagIDs []string) (*model.Link, error) {
	if url == "" {
		return nil, errors.New("url is required")
	}

	linkID, err := s.repo.GetOrCreateLink(url)
	if err != nil {
		return nil, err
	}

	err = s.repo.ReplaceLinkTags(linkID, tagIDs)
	if err != nil {
		return nil, err
	}

	// Получить обновлённый линк с тегами
	links, err := s.repo.GetAllLinksWithTags()
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if link.ID == linkID {
			return &link, nil
		}
	}

	return nil, errors.New("link not found after creation")
}

func (s *LinkService) GetAllLinks() ([]model.Link, error) {
	return s.repo.GetAllLinksWithTags()
}

func (s *LinkService) GetLinksByTags(tags []string) ([]model.Link, error) {
	return s.repo.GetLinksByTags(tags)
}

func (s *LinkService) DeleteLink(id string) error {
	if id == "" {
		return errors.New("link id is required")
	}
	return s.repo.DeleteLink(id)
}
