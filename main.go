package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := loadToken()

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session", err)
		return
	}

	dg.AddHandler(messageCreate)
	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

type config struct {
	Token       string `json:"token"`
	Permissions int    `json:"permissions"`
}

func loadToken() string {
	file, err := os.Open("config/config.json")
	if err != nil {
		fmt.Println("could not open config file: ", err)
	}

	defer file.Close()
	decoder := json.NewDecoder(file)
	config := config{}

	if err := decoder.Decode(&config); err != nil {
		fmt.Println("could not read config: ", err)
	}
	return config.Token
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	buyPriceMsg := regexp.MustCompile("")
	sellPriceMsg := regexp.MustCompile("")

	switch {
	case buyPriceMsg.MatchString(m.Content):
		s.ChannelMessageSend(m.ChannelID, "Got buy price")
	case sellPriceMsg.MatchString(m.Content):
		s.ChannelMessageSend(m.ChannelID, "Got sell price")
	}
}
