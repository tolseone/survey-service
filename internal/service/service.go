package service

import (
	"github.com/mattermost/mattermost-server/v6/model"
)

type Service struct {
	Client *model.Client4
}

func NewService(client *model.Client4) *Service {
	return &Service{
		Client: client,
	}
}
