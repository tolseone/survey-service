package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"

	appl "survey/internal/app"
	cfg "survey/internal/config"
	http "survey/internal/http"
	service "survey/internal/service"
)

func main() {
	// Init logger
	logger := zerolog.New(
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC822,
		},
	).With().Timestamp().Logger()

	logger.Info().Msg("initialized logger")

	// Load config
	config := cfg.LoadConfig()

	logger.Info().Msg("loaded config")

	logger.Info().Str("MattermostServer", config.MattermostServer.String()).Msg("Loaded Mattermost server URL")
	logger.Info().Str("MattermostToken", config.MattermostToken).Msg("Loaded Mattermost token")
	// Create MM client
	mattermostClient := model.NewAPIv4Client(config.MattermostServer.String())
	mattermostClient.SetToken(config.MattermostToken)

	logger.Info().Msg("Created MM client")

	// Create Websocket client
	mattermostWebSocketClient, err := model.NewWebSocketClient4(
		fmt.Sprintf("ws://%s", config.MattermostServer.Host+config.MattermostServer.Path),
		mattermostClient.AuthToken,
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create Websocket client")
	}
	defer mattermostWebSocketClient.Close()

	logger.Info().Msg("Created Websocket client")

	// Create service
	srv := service.NewService(mattermostClient)

	// Create app
	app := appl.NewApplication(config, logger, mattermostClient, mattermostWebSocketClient, srv)

	// Create http-server
	serverHTTP := http.NewServer(app)

	// Setup Graceful shutdown
	setupGracefulShutdown(logger, serverHTTP)

	// Connect to MM server
	if user, _, err := mattermostClient.GetUser("me", ""); err != nil {
		logger.Fatal().Err(err).Msg("Failed to log in to Mattermost")
	} else {
		logger.Info().Msg("Logged in to Mattermost")
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
	appl.SendMsgToTalkingChannel(app, "Hi! I am a bot.", "")

	// Listen to live events coming in via websocket in a separate goroutine.
	go appl.ListenToEvents(app)

	// Start HTTP-server
	logger.Info().Msg("Starting HTTP server")
	go func() {
		err := serverHTTP.Server.ListenAndServe()
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Wait for termination signal
	<-make(chan os.Signal, 1)
}

func setupGracefulShutdown(logger zerolog.Logger, server *http.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		logger.Info().Msg("Shutting down HTTP server")
		server.Server.Close()
	}()
}
