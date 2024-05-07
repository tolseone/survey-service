package app

import (
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"
	cfg "survey/internal/config"
)

// application struct to hold the dependencies for our bot
type Application struct {
	Config                    cfg.Config
	Logger                    zerolog.Logger
	MattermostClient          *model.Client4
	MattermostWebSocketClient *model.WebSocketClient
	MattermostUser            *model.User
	MattermostChannel         *model.Channel
	MattermostTeam            *model.Team
}
