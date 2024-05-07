package service

import (
	"github.com/mattermost/mattermost-server/v6/model"
)

type Repository interface {
}

type Service struct {
	client *model.Client4
}

func NewService(client *model.Client4) *Service {
	return &Service{
		client: client,
	}
}
