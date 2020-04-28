package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	sqlUserTableName   = "user"
	sqlUserID          = "user_id"
	sqlUserDiscordID   = "discord_user_id"
	sqlUserDiscordName = "discord_user_name"
	sqlWeekTableName   = "week"
	sqlWeekID          = "week_id"
	sqlWeekStartDay    = "week_start_day"
	sqlWeekStartPrice  = "week_start_price"
	sqlPriceTableName  = "price"
	sqlPriceID         = "price_id"
	sqlPriceDay        = "price_day"
	sqlPriceTime       = "price_time"
	sqlPriceSell       = "price_sell"
	timeFormat         = "2006-01-02"
)

// TurnipAccess stores connection information for the turnip db.
type TurnipAccess struct {
	db *sql.DB
}

// NewTurnipAccess creates a database access object for the turnip db.
func NewTurnipAccess() (*TurnipAccess, error) {
	access := &TurnipAccess{}
	var err error
	access.db, err = sql.Open("sqlite3", "./turnips.db")
	if err != nil {
		return nil, err
	}

	return access, nil
}

// CreateTables creates all of the tables in the turnip tb.
func (a *TurnipAccess) CreateTables() error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	createUserTable, err := tx.Prepare(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s INTEGER NOT NULL PRIMARY KEY, %s TEXT NOT NULL, %s TEXT NOT NULL);`, sqlUserTableName, sqlUserID, sqlUserDiscordID, sqlUserDiscordName))
	if err != nil {
		tx.Rollback()
		return err
	}

	createWeekTable, err := tx.Prepare(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s INTEGER NOT NULL PRIMARY KEY, %s INTEGER NOT NULL, %s TEXT NOT NULL, %s INTEGER NOT NULL, FOREIGN KEY(%s) REFERENCES %s(%s));`, sqlWeekTableName, sqlWeekID, sqlUserID, sqlWeekStartDay, sqlWeekStartPrice, sqlUserID, sqlUserTableName, sqlUserID))
	if err != nil {
		tx.Rollback()
		return err
	}

	createPriceTable, err := tx.Prepare(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s INTEGER NOT NULL PRIMARY KEY, %s INTEGER NOT NULL, %s INTEGER NOT NULL, %s TEXT NOT NULL, %s INTEGER NOT NULL, FOREIGN KEY(%s) REFERENCES %s(%s));`, sqlPriceTableName, sqlPriceID, sqlWeekID, sqlPriceDay, sqlPriceTime, sqlPriceSell, sqlWeekID, sqlWeekTableName, sqlWeekID))
	if err != nil {
		tx.Rollback()
		return err
	}

	if _, err := createUserTable.Exec(); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := createWeekTable.Exec(); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := createPriceTable.Exec(); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

type weekObject struct {
	id         int
	userID     int
	startDay   string
	startPrice int
}

func getNearestSunday(day time.Time) (time.Time, error) {
	wDay := day.Weekday()
	if wDay != time.Sunday {
		day = day.AddDate(0, 0, -int(wDay)) // Subtract days past Sunday to get most recent Sunday.
	}

	return day, nil
}

// GetWeek checks the db if week exists. If one is found, then the weekID is returned
func (a *TurnipAccess) GetWeek(userID int, day time.Time) (int, error) {
	var err error
	day, err = getNearestSunday(day)
	if err != nil {
		return 0, err
	}
	sunday := day.Format(timeFormat)

	// Prepare the query to check if the price exists already
	checkWeek, err := a.db.Prepare(fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ? AND %s = ?;`, sqlWeekID, sqlWeekTableName, sqlUserID, sqlWeekStartDay))
	if err != nil {
		return 0, err
	}

	// Check if the price is already provided and update if so
	row := checkWeek.QueryRow(userID, sunday)
	var id64 int64
	if err := row.Scan(&id64); err != nil {
		return 0, err
	}

	return int(id64), nil
}

// CreateOrUpdateWeek checks the db if the week exists or creates a new week if not. In both cases with no error, the week ID is returned.
func (a *TurnipAccess) CreateOrUpdateWeek(userID int, day time.Time, price int) (int, error) {
	sundayTime, err := getNearestSunday(day)
	if err != nil {
		return 0, err
	}
	sunday := sundayTime.Format(timeFormat)

	weekID, err := a.GetWeek(userID, day)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Prepare the query to insert the price
			insertSellPrice, err := a.db.Prepare(fmt.Sprintf(`INSERT INTO %s (%s, %s, %s) VALUES (?, ?, ?);`, sqlWeekTableName, sqlUserID, sqlWeekStartDay, sqlWeekStartPrice))
			if err != nil {
				return 0, err
			}

			// Run the prepared query
			res, err := insertSellPrice.Exec(userID, sunday, price)
			if err != nil {
				return 0, err
			}

			// Get the ID of the price
			id64, err := res.LastInsertId()
			if err != nil {
				return 0, err
			}

			return int(id64), nil
		}
		return 0, err
	}

	// Update the sell price on the existing week
	updateSellPrice, err := a.db.Prepare(fmt.Sprintf(`UPDATE %s SET %s = ? WHERE %s = ?;`, sqlWeekTableName, sqlWeekStartPrice, sqlWeekID))
	if err != nil {
		return 0, err
	}

	// Run the prepared query
	if _, err := updateSellPrice.Exec(price, weekID); err != nil {
		return 0, err
	}

	return weekID, nil
}

type priceObject struct {
	id     int
	weekID int
	day    int
	time   string
	sell   int
}

// CreateOrUpdateBuyPrice checks the db if the price exists or creates a new price if not.
// If the price exists and the values are different, the price is updated.
func (a *TurnipAccess) CreateOrUpdateBuyPrice(weekID int, dayNum int, meridian string, price int) (int, error) {
	// Validate the meridian string
	if meridian != "am" && meridian != "pm" {
		return 0, fmt.Errorf("string \"%s\" must be am or pm", meridian)
	}

	// Prepare the query to check if the price exists already
	checkBuyPrice, err := a.db.Prepare(fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ? AND %s = ? AND %s = ?;`, sqlPriceID, sqlPriceTableName, sqlWeekID, sqlPriceDay, sqlPriceTime))
	if err != nil {
		return 0, err
	}

	// Check if the price is already provided and update if so
	row := checkBuyPrice.QueryRow(weekID, dayNum, meridian)
	var id64 int64
	if err := row.Scan(&id64); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Prepare the query to insert the price
			insertBuyPrice, err := a.db.Prepare(fmt.Sprintf(`INSERT INTO %s (%s, %s, %s, %s) VALUES (?, ?, ?, ?);`, sqlPriceTableName, sqlWeekID, sqlPriceDay, sqlPriceTime, sqlPriceSell))
			if err != nil {
				return 0, err
			}

			// Run the prepared query
			res, err := insertBuyPrice.Exec(weekID, dayNum, meridian, price)
			if err != nil {
				return 0, err
			}

			// Get the ID of the price
			id64, err := res.LastInsertId()
			if err != nil {
				return 0, err
			}

			return int(id64), nil
		}
		return 0, err

	}

	// Prepare the query to update the price
	updateBuyPrice, err := a.db.Prepare(fmt.Sprintf(`UPDATE %s SET %s = ? WHERE %s = ?;`, sqlPriceTableName, sqlPriceSell, sqlPriceID))
	if err != nil {
		return 0, err
	}

	// Run the prepared query
	if _, err := updateBuyPrice.Exec(price, id64); err != nil {
		return 0, err
	}

	return int(id64), nil
}

// GetOrCreateUser checks the db if the user exists or creates a new user if not. In both cases with no error, the user ID is returned.
func (a *TurnipAccess) GetOrCreateUser(discordUserID string, discordUserName string) (int, error) {

	// Prepare the query to check if the user exists already
	checkUser, err := a.db.Prepare(fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ?;`, sqlUserID, sqlUserTableName, sqlUserDiscordID))
	if err != nil {
		return 0, err
	}

	// Check if the user is already registered and exit if so
	row := checkUser.QueryRow(discordUserID)
	var id int
	if err := row.Scan(&id); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	// User is already registered
	if id > 0 {
		return id, nil
	}

	// Prepare the query to save the new user
	insertUser, err := a.db.Prepare(fmt.Sprintf(`INSERT INTO %s (%s, %s) VALUES (?, ?);`, sqlUserTableName, sqlUserDiscordID, sqlUserDiscordName))
	if err != nil {
		return 0, err
	}

	// Run the prepared query
	res, err := insertUser.Exec(discordUserID, discordUserName)
	if err != nil {
		return 0, err
	}

	// Get the ID of the user
	id64, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id64), nil
}

// Close closes the stored db connection.
func (a *TurnipAccess) Close() error {
	return a.db.Close()
}
