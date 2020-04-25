package main

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	access, err := NewTurnipAccess()
	if err != nil {
		fmt.Println("error creating SQLite tables", err)
		return
	}
	defer access.Close()

	dg, err := initDiscord()
	if err != nil {
		fmt.Println("error creating Discord session", err)
		return
	}
	// Wait here until CTRL-C or other term signal is received
	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()

	// Cleanly close down the Discord session.
	dg.Close()
}

func initDiscord() (*discordgo.Session, error) {
	token := loadToken()
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session", err)
		return nil, err
	}

	dg.AddHandler(messageCreate)
	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return nil, err
	}
	return dg, nil
}

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
	createUserTable := `CREATE TABLE IF NOT EXISTS user (user_id INTEGER NOT NULL PRIMARY KEY, discord_user_id TEXT NOT NULL, user_name TEXT NOT NULL);`
	createWeekTable := `CREATE TABLE IF NOT EXISTS week (week_id INTEGER NOT NULL PRIMARY KEY, user_id INTEGER NOT NULL, start_day TEXT NOT NULL, buy_price INTEGER NOT NULL, FOREIGN KEY(userid)) REFERENCES user(id);`
	createPriceTable := `CREATE TABLE IF NOT EXISTS price (price_id INTEGER NOT NULL PRIMARY KEY, day INTEGER NOT NULL, time TEXT NOT NULL, sell_price INTEGER NOT NULL);`

	_, err := a.db.Exec(fmt.Sprint(createUserTable, createWeekTable, createPriceTable))
	if err != nil {
		return err
	}
	return nil
}

// Close closes the stored db connection.
func (a *TurnipAccess) Close() error {
	return a.db.Close()
}

func initSQLite(db *sql.DB) error {

	// tx, err := db.Begin()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// stmt, err := tx.Prepare("insert into foo(id, name) values(?, ?)")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer stmt.Close()
	// for i := 0; i < 100; i++ {
	// 	_, err = stmt.Exec(i, fmt.Sprintf("こんにちわ世界%03d", i))
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
	// tx.Commit()

	// rows, err := db.Query("select id, name from foo")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer rows.Close()
	// for rows.Next() {
	// 	var id int
	// 	var name string
	// 	err = rows.Scan(&id, &name)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Println(id, name)
	// }
	// err = rows.Err()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// stmt, err = db.Prepare("select name from foo where id = ?")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer stmt.Close()
	// var name string
	// err = stmt.QueryRow("3").Scan(&name)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(name)

	// _, err = db.Exec("delete from foo")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// _, err = db.Exec("insert into foo(id, name) values(1, 'foo'), (2, 'bar'), (3, 'baz')")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// rows, err = db.Query("select id, name from foo")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer rows.Close()
	// for rows.Next() {
	// 	var id int
	// 	var name string
	// 	err = rows.Scan(&id, &name)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Println(id, name)
	// }
	// err = rows.Err()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	return nil
}
