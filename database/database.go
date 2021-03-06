package database

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pagarme/teleport/action"
	"github.com/pagarme/teleport/asset"
	"log"
)

// Database definition
type Database struct {
	Name     string
	Database string
	Hostname string
	Username string
	Password string
	Port     int
	Schemas  map[string]*Schema
	Db       *sqlx.DB
}

func New(name, database, hostname, username, password string, port int) *Database {
	return &Database{
		Name:     name,
		Database: database,
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     port,
		Schemas:  make(map[string]*Schema),
	}
}

// Open connection with database and setup internal tables
func (db *Database) Start() error {
	var err error

	db.Db, err = sqlx.Connect("postgres", fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		db.Hostname,
		db.Username,
		db.Password,
		db.Database,
		db.Port,
	))

	db.Db.SetMaxOpenConns(5)

	if err != nil {
		return err
	}

	err = db.setupTables()

	if err != nil {
		return err
	}

	err = db.RefreshSchema()

	if err != nil {
		return err
	}

	return nil
}

func (db *Database) NewTransaction() *sqlx.Tx {
	return db.Db.MustBegin()
}

// Install triggers on a source table
func (db *Database) InstallTriggers(targetExpression string) error {
	err := db.installDDLTriggers()

	if err != nil {
		return err
	}

	// Install triggers for each table
	for _, schema := range db.Schemas {
		for _, class := range schema.Tables {
			// If class is not a table, continue...
			if class.RelationKind != "r" {
				continue
			}

			if action.IsInTargetExpression(&targetExpression, &schema.Name, &class.RelationName) {
				class.InstallTriggers()
				//
				// if err != nil {
				// 	return err
				// }
			}
		}
	}

	return nil
}

// Install triggers in schema
func (db *Database) installDDLTriggers() error {
	rows, err := db.runQueryFromFile("data/sql/source_trigger.sql")

	if err == nil {
		log.Printf("Installed triggers on database")
	} else {
		log.Printf("Failed to install triggers on database: %v", err)
	}

	rows.Close()
	return err
}

// Setup internal tables using setup script
func (db *Database) setupTables() error {
	rows, err := db.runQueryFromFile("data/sql/setup.sql")

	if err != nil {
		return err
	}

	rows.Close()
	return nil
}

// Run query on database
func (db *Database) runQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return db.Db.Query(query, args...)
}

func (db *Database) selectObjs(v interface{}, query string, args ...interface{}) error {
	return db.Db.Select(v, query, args...)
}

// Open file and run query
func (db *Database) runQueryFromFile(file string) (*sql.Rows, error) {
	// Read file
	content, err := asset.Asset(file)

	if err != nil {
		return nil, err
	}

	return db.runQuery(string(content))
}
