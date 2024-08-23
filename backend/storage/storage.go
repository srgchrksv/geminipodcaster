package storage

import "github.com/srgchrksv/geminipodcaster/models"


type Storage struct {
	PodcastSessions map[string]*models.PodcastSession
}

func NewStorage() *Storage {
	return &Storage{
		PodcastSessions: make(map[string]*models.PodcastSession),
	}
}
