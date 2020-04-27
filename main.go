package main

import (
	"fmt"
	"sync"
)

func main() {
	access, err := NewTurnipAccess()
	if err != nil {
		fmt.Println("error creating access:", err)
		return
	}
	defer access.Close()

	if err := access.CreateTables(); err != nil {
		fmt.Println("error creating tables:", err)
		return
	}

	bot, err := NewDiscordBot(access)
	if err != nil {
		fmt.Println("error creating Discord session:", err)
		return
	}
	defer bot.Close() // close the DiscordBot session

	// Wait here until CTRL-C or other term signal is received
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
