package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type discordConf struct {
	Token       string `json:"token"`
	Permissions int    `json:"permissions"`
}

type config struct {
	Discord discordConf `json:"discord"`
}

// DiscordBot controls the turnip bot and stores all required connection details.
type DiscordBot struct {
	access *TurnipAccess
	dg     *discordgo.Session
}

const chartBaseURL = "https://ac-turnip.com/share?f="

// NewDiscordBot creates an active session for a discord bot.
func NewDiscordBot(access *TurnipAccess) (*DiscordBot, error) {
	token := loadToken()
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		errors.Wrap(err, "error creating Discord session:")
		return nil, err
	}

	var bot = &DiscordBot{dg: dg, access: access}

	// Add handlers for the different events
	dg.AddHandler(bot.messageCreateHook())

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		errors.Wrap(err, "error opening connection:")
		return nil, err
	}

	fmt.Println("turnip bot has started, awaiting messages")
	return bot, nil
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
	return config.Discord.Token
}

func (d *DiscordBot) messageCreateHook() func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		// This isn't required in this specific example but it's a good practice.
		if m.Author.ID == s.State.User.ID {
			return
		}

		buyCmd := regexp.MustCompile(`^!buy\s*\d+\s*([mM]onday|[tT]uesday|[wW]ednesday|[tT]hursday|[fF]riday|[sS]aturday)?\s*(am|pm)?`)
		sellCmd := regexp.MustCompile(`^!sell\s*\d+`)
		chartCmd := regexp.MustCompile(`^!chart`)

		switch {
		case buyCmd.MatchString(m.Content):
			err := d.buyCommand(s, m)
			if err != nil {
				fmt.Println("buy command: ", err)
				return
			}
		case sellCmd.MatchString(m.Content):
			err := d.sellCommand(s, m)
			if err != nil {
				fmt.Println("sell command: ", err)
				return
			}
		case chartCmd.MatchString(m.Content):
			err := d.chartCommand(s, m)
			if err != nil {
				fmt.Println("chart command: ", err)
				return
			}
		}
	}
}

func (d *DiscordBot) buyCommand(s *discordgo.Session, m *discordgo.MessageCreate) error {
	priceArg := regexp.MustCompile(`\d+`)
	dayArg := regexp.MustCompile(`\b([mM]onday|[tT]uesday|[wW]ednesday|[tT]hursday|[fF]riday|[sS]aturday)\b`)
	meridianArg := regexp.MustCompile(`\b(am|pm)\b`)

	// Parse price
	rawPrice := priceArg.FindString(m.Content)
	price, err := strconv.Atoi(rawPrice)
	if err != nil {
		return fmt.Errorf("unable to parse price: %w", err)
	}

	// Parse date
	messageCreateTime, err := snowflakeCreationTime(m.ID) // Get the day based on message creation time
	if err != nil {
		return fmt.Errorf("unable to get message timestamp: %w", err)
	}

	day := dayArg.FindString(m.Content)                      // Parse the day argument
	weekdayNum, err := formatWeekday(day, messageCreateTime) // Get the day from the argument or messageCreateTime
	if err != nil {
		return err
	}

	// parse am/pm
	meridian := meridianArg.FindString(m.Content)

	// Check if user already exists
	userID, err := d.access.GetOrCreateUser(m.Author.ID, m.Author.Username)
	if err != nil {
		return fmt.Errorf("unable to get user registry: %w", err)
	}

	// Check if week already exists
	week, err := d.access.GetWeekForUser(userID, messageCreateTime)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s please provide your sell price for the week before posting buy prices.", m.Author.Mention()))
		return fmt.Errorf("unable to get week for user: %w", err)
	}

	// Create or update the buy price
	if _, err := d.access.CreateOrUpdateBuyPrice(week.ID, weekdayNum, meridian, price); err != nil {
		return fmt.Errorf("unable to save buy for week: %w", err)
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🤖")
	fmt.Println(fmt.Sprintf("got %s %s buy price %d for user %s", time.Weekday(weekdayNum).String(), meridian, price, m.Author.Username))
	return nil
}

func (d *DiscordBot) sellCommand(s *discordgo.Session, m *discordgo.MessageCreate) error {
	priceArg := regexp.MustCompile(`\d+`)

	// Parse price
	rawPrice := priceArg.FindString(m.Content)
	price, err := strconv.Atoi(rawPrice)
	if err != nil {
		return fmt.Errorf("unable to parse price: %w", err)
	}

	// Parse date
	messageCreateTime, err := snowflakeCreationTime(m.ID) // Get the day based on message creation time
	if err != nil {
		return fmt.Errorf("unable to get message timestamp: %w", err)
	}

	// Check if user already exists
	userID, err := d.access.GetOrCreateUser(m.Author.ID, m.Author.Username)
	if err != nil {
		return fmt.Errorf("unable to get user registry: %w", err)
	}

	// Create or update the week (sell price)
	if _, err := d.access.CreateOrUpdateWeek(userID, messageCreateTime, price); err != nil {
		return fmt.Errorf("unable to create week for user: %w", err)
	}

	fmt.Println(fmt.Sprintf("got sell price %d for user %s", price, m.Author.Username))
	s.MessageReactionAdd(m.ChannelID, m.ID, "🤖")
	return nil
}

func (d *DiscordBot) chartCommand(s *discordgo.Session, m *discordgo.MessageCreate) error {
	// Parse date
	messageCreateTime, err := snowflakeCreationTime(m.ID) // Get the day based on message creation time
	if err != nil {
		return fmt.Errorf("unable to get message timestamp: %w", err)
	}

	// Check if user already exists
	userID, err := d.access.GetOrCreateUser(m.Author.ID, m.Author.Username)
	if err != nil {
		return fmt.Errorf("unable to get user registry: %w", err)
	}

	// Check if week already exists
	week, err := d.access.GetWeekForUser(userID, messageCreateTime)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s you must provide your sell price for the week before a chart can be made.", m.Author.Mention()))
		return fmt.Errorf("unable to get week for user: %w", err)
	}

	prices, err := d.access.GetBuyPricesForWeek(week.ID)
	if err != nil {
		return fmt.Errorf("unable to buy prices for week: %w", err)
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🤖")
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Here is a chart of your prices so far this week: %s", getPriceURL(week.StartPrice, prices)))
	return nil
}

// Close closes the discord connections on the bot.
func (d *DiscordBot) Close() error {
	return d.dg.Close()
}

// snowflakeCreationTime extracts the timestamp from the snowflake (discord ID system)
func snowflakeCreationTime(ID string) (t time.Time, err error) {
	i, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return
	}
	timestamp := (i >> 22) + 1420070400000
	t = time.Unix(timestamp/1000, 0)
	return
}

var daysOfWeek = map[string]time.Weekday{
	"Sunday":    time.Sunday,
	"Monday":    time.Monday,
	"Tuesday":   time.Tuesday,
	"Wednesday": time.Wednesday,
	"Thursday":  time.Thursday,
	"Friday":    time.Friday,
	"Saturday":  time.Saturday,
}

func parseWeekday(v string) (time.Weekday, error) {
	if d, ok := daysOfWeek[v]; ok {
		return d, nil
	}

	return time.Sunday, fmt.Errorf("invalid weekday '%s'", v)
}

func formatWeekday(day string, referenceTime time.Time) (time.Weekday, error) {
	if day == "" {
		return referenceTime.Weekday(), nil // If no day provided, infer weekday from the reference time
	}

	day = strings.Title(strings.ToLower(day)) // Capitalize first letter
	weekday, err := parseWeekday(day)         // Get the numeric value of the weekday
	if err != nil {
		return 0, err
	}
	return weekday, nil
}

func getPriceURL(sellPrice int, buyPrices []Price) string {
	weekPrices := make([]string, 13) // One Sunday price and 2 prices for each other day of the week
	for i := range weekPrices {
		dayTime := i / 2
		price := "0"
		if i%2 == 1 { // Odd (morning price)
			dayTime++ // Correct rounding
			found, err := findPrice(buyPrices, "am", time.Weekday(dayTime))
			if err == nil {
				price = strconv.Itoa(found)
			}
		} else { // Even (pm price)
			found, err := findPrice(buyPrices, "pm", time.Weekday(dayTime))
			if err == nil {
				price = strconv.Itoa(found)
			}
		}
		weekPrices[i] = price // Assign the price if found
	}
	weekPrices[0] = strconv.Itoa(sellPrice) // Assign the Sunday price
	fmt.Println("prices: ", weekPrices)

	return chartBaseURL + strings.Join(weekPrices, "-")
}

func findPrice(prices []Price, time string, day time.Weekday) (int, error) {
	for _, b := range prices {
		if b.Day == day && b.Time == time {
			return b.Sell, nil // Found the price
		}
	}
	return 0, errors.New("no matching price found") // No price found
}
