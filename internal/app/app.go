package app

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"

	cfg "survey/internal/config"
	service "survey/internal/service"
)

type Application struct {
	Config                    cfg.Config
	Logger                    zerolog.Logger
	MattermostClient          *model.Client4
	MattermostWebSocketClient *model.WebSocketClient
	MattermostUser            *model.User
	MattermostChannel         *model.Channel
	MattermostTeam            *model.Team
	Service                   *service.Service
}

func NewApplication(config cfg.Config, logger zerolog.Logger, client *model.Client4, websocketClient *model.WebSocketClient, service *service.Service) *Application {
	return &Application{
		Config:                    config,
		Logger:                    logger,
		MattermostClient:          client,
		MattermostWebSocketClient: websocketClient,
		MattermostUser:            &model.User{},
		MattermostChannel:         &model.Channel{},
		MattermostTeam:            &model.Team{},
		Service:                   service,
	}
}

func SendMsgToTalkingChannel(app *Application, msg string, replyToId string) {
	// Note that replyToId should be empty for a new post.
	// All replies in a thread should reply to root.

	post := &model.Post{}
	post.ChannelId = app.MattermostChannel.Id
	post.Message = msg

	post.RootId = replyToId

	if _, _, err := app.MattermostClient.CreatePost(post); err != nil {
		app.Logger.Error().Err(err).Str("RootID", replyToId).Msg("Failed to create post")
	}
}

func ListenToEvents(app *Application) {
	var err error
	failCount := 0
	for {
		app.MattermostWebSocketClient, err = model.NewWebSocketClient4(
			fmt.Sprintf("ws://%s", app.Config.MattermostServer.Host+app.Config.MattermostServer.Path),
			app.MattermostClient.AuthToken,
		)
		if err != nil {
			app.Logger.Warn().Err(err).Msg("Mattermost websocket disconnected, retrying")
			failCount += 1
			// TODO: backoff based on failCount and sleep for a while.
			continue
		}
		app.Logger.Info().Msg("Mattermost websocket connected")

		app.MattermostWebSocketClient.Listen()

		for event := range app.MattermostWebSocketClient.EventChannel {
			// Launch new goroutine for handling the actual event.
			// If required, you can limit the number of events beng processed at a time.
			go handleWebSocketEvent(app, event)
		}
	}
}

func handleWebSocketEvent(app *Application, event *model.WebSocketEvent) {

	// Ignore other channels.
	if event.GetBroadcast().ChannelId != app.MattermostChannel.Id {
		return
	}

	// Ignore other types of events.
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	// Since this event is a post, unmarshal it to (*model.Post)
	post := &model.Post{}
	err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &post)
	if err != nil {
		app.Logger.Error().Err(err).Msg("Could not cast event to *model.Post")
	}

	// Ignore messages sent by this bot itself.
	if post.UserId == app.MattermostUser.Id {
		return
	}

	// Handle however you want.
	handlePost(app, post)
}

func handlePost(app *Application, post *model.Post) {
	app.Logger.Debug().Str("message", post.Message).Msg("")
	app.Logger.Debug().Interface("post", post).Msg("")

	if matched, _ := regexp.MatchString(`(?:^|\W)hello(?:$|\W)`, post.Message); matched {

		// If post has a root ID then its part of thread, so reply there.
		// If not, then post is independent, so reply to the post.
		if post.RootId != "" {
			SendMsgToTalkingChannel(app, "I replied in an existing thread.", post.RootId)
		} else {
			SendMsgToTalkingChannel(app, "I just replied to a new post, starting a chain.", post.Id)
		}
		return
	}
}
