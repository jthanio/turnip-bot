package main

import (
	"fmt"
	"sync"

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

	// Wait here until CTRL-C or other term signal is received
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()

	// Cleanly close down the Discord session.
	dg.Close()
}
