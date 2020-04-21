package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

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

	buyCmd := regexp.MustCompile(`^!buy`)
	sellCmd := regexp.MustCompile(`^!sell`)
	priceRE := regexp.MustCompile(`\d+`)

	switch {
	case buyCmd.MatchString(m.Content):
		test := priceRE.FindString(m.Content)
		fmt.Println(test)
		price, err := strconv.Atoi(test)
		if err != nil {
			fmt.Println("unable to parse integer: ", err)
		}
		fmt.Println(fmt.Sprintf("got buy price %d", price))
		s.MessageReactionAdd(m.ChannelID, m.ID, "🤖")
		//s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Got buy price %d", price))
	case sellCmd.MatchString(m.Content):
		test := priceRE.FindString(m.Content)
		fmt.Println(test)
		price, _ := strconv.Atoi(test)
		fmt.Println(fmt.Sprintf("got sell price %d", price))
		s.MessageReactionAdd(m.ChannelID, m.ID, "🤖")
		//s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Got sell price: %d", price))
	}
}
