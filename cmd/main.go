package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"
	appl "survey/internal/app"
	cfg "survey/internal/config"
)

func main() {

	app := &appl.Application{
		Logger: zerolog.New(
			zerolog.ConsoleWriter{
				Out:        os.Stdout,
				TimeFormat: time.RFC822,
			},
		).With().Timestamp().Logger(),
	}

	app.Config = cfg.LoadConfig()
	app.Logger.Info().Str("config", fmt.Sprint(app.Config)).Msg("")

	setupGracefulShutdown(app)

	// Create a new mattermost client.
	app.MattermostClient = model.NewAPIv4Client(app.Config.MattermostServer.String())

	// Login.
	app.MattermostClient.SetToken(app.Config.MattermostToken)

	if user, resp, err := app.MattermostClient.GetUser("me", ""); err != nil {
		app.Logger.Fatal().Err(err).Msg("Could not log in")
	} else {
		app.Logger.Debug().Interface("user", user).Interface("resp", resp).Msg("")
		app.Logger.Info().Msg("Logged in to mattermost")
		app.MattermostUser = user
	}

	// Find and save the bot's team to app struct.
	if team, resp, err := app.MattermostClient.GetTeamByName(app.Config.MattermostTeamName, ""); err != nil {
		app.Logger.Fatal().Err(err).Msg("Could not find team. Is this bot a member ?")
	} else {
		app.Logger.Debug().Interface("team", team).Interface("resp", resp).Msg("")
		app.MattermostTeam = team
	}

	// Find and save the talking channel to app struct.
	if channel, resp, err := app.MattermostClient.GetChannelByName(
		app.Config.MattermostChannel, app.MattermostTeam.Id, "",
	); err != nil {
		app.Logger.Fatal().Err(err).Msg("Could not find channel. Is this bot added to that channel ?")
	} else {
		app.Logger.Debug().Interface("channel", channel).Interface("resp", resp).Msg("")
		app.MattermostChannel = channel
	}

	// Send a message (new post).
	sendMsgToTalkingChannel(app, "Hi! I am a bot.", "")

	// Listen to live events coming in via websocket.
	listenToEvents(app)
}

func setupGracefulShutdown(app *appl.Application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if app.MattermostWebSocketClient != nil {
				app.Logger.Info().Msg("Closing websocket connection")
				app.MattermostWebSocketClient.Close()
			}
			app.Logger.Info().Msg("Shutting down")
			os.Exit(0)
		}
	}()
}

func sendMsgToTalkingChannel(app *appl.Application, msg string, replyToId string) {
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

func listenToEvents(app *appl.Application) {
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

func handleWebSocketEvent(app *appl.Application, event *model.WebSocketEvent) {

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

func handlePost(app *appl.Application, post *model.Post) {
	app.Logger.Debug().Str("message", post.Message).Msg("")
	app.Logger.Debug().Interface("post", post).Msg("")

	if matched, _ := regexp.MatchString(`(?:^|\W)hello(?:$|\W)`, post.Message); matched {

		// If post has a root ID then its part of thread, so reply there.
		// If not, then post is independent, so reply to the post.
		if post.RootId != "" {
			sendMsgToTalkingChannel(app, "I replied in an existing thread.", post.RootId)
		} else {
			sendMsgToTalkingChannel(app, "I just replied to a new post, starting a chain.", post.Id)
		}
		return
	}
}
