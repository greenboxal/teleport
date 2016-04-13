package database

import (
	"encoding/gob"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pagarme/teleport/config"
	"os"
	"reflect"
	"testing"
)

type StubAction struct {
	Name string
}

func (s *StubAction) Execute(tx *sqlx.Tx) error {
	return nil
}

func (s *StubAction) Filter(targetExpression string) bool {
	return true
}

var db *Database
var stubAction *StubAction
var stubActionData string

func init() {
	gob.Register(&StubAction{})

	config := config.New()
	err := config.ReadFromFile("../config_test.yml")

	if err != nil {
		fmt.Printf("Error opening config file: %v\n", err)
		os.Exit(1)
	}

	db = New(
		config.Database.Name,
		config.Database.Database,
		config.Database.Hostname,
		config.Database.Username,
		config.Database.Password,
		config.Database.Port,
	)

	// Start db
	if err = db.Start(); err != nil {
		fmt.Printf("Erro starting database: ", err)
		os.Exit(1)
	}

	stubAction = &StubAction{"action data"}
	stubActionData = "OBAAFCpkYXRhYmFzZS5TdHViQWN0aW9u/4EDAQEKU3R1YkFjdGlvbgH/ggABAQEETmFtZQEMAAAAEf+CDgELYWN0aW9uIGRhdGEA"
}

func TestNewEvent(t *testing.T) {
	event := NewEvent("a,b,c,d,e,f")

	data := "f"

	testEvent := &Event{
		Id:            "a",
		Kind:          "b",
		Status:        "",
		TriggerTag:    "c",
		TriggerEvent:  "d",
		TransactionId: "e",
		Data:          &data,
	}

	if !reflect.DeepEqual(event, testEvent) {
		t.Errorf(
			"new event => %#v, want %#v",
			event,
			testEvent,
		)
	}
}

func TestGetEvents(t *testing.T) {
	db.db.Exec(`
		TRUNCATE teleport.event;
		INSERT INTO teleport.event
			(id, kind, status, trigger_tag, trigger_event, transaction_id, data)
			VALUES
			(1, 'ddl', 'waiting_batch', '123', 'event', '456', 'asd');
		INSERT INTO teleport.event
			(id, kind, status, trigger_tag, trigger_event, transaction_id, data)
			VALUES
			(2, 'ddl', 'building', '123', 'event', '456', 'asd');
	`)

	data := "asd"

	testEvent := Event{
		Id:            "1",
		Kind:          "ddl",
		Status:        "waiting_batch",
		TriggerTag:    "123",
		TriggerEvent:  "event",
		TransactionId: "456",
		Data:          &data,
	}

	events, err := db.GetEvents("waiting_batch")

	if err != nil {
		t.Errorf("get events returned error: %v\n", err)
	}

	if len(events) != 1 {
		t.Errorf("get events => %d, want %d", len(events), 1)
	}

	if !reflect.DeepEqual(events[0], testEvent) {
		t.Errorf(
			"get events => %#v, want %#v",
			events[0],
			testEvent,
		)
	}
}

func TestGetEvent(t *testing.T) {
	db.db.Exec(`
		TRUNCATE teleport.event;
		INSERT INTO teleport.event
			(id, kind, status, trigger_tag, trigger_event, transaction_id, data)
			VALUES
			(1, 'ddl', 'waiting_batch', '123', 'event', '456', 'asd_one');
		INSERT INTO teleport.event
			(id, kind, status, trigger_tag, trigger_event, transaction_id, data)
			VALUES
			(2, 'ddl', 'building', '123', 'event', '456', 'asd');
	`)

	data := "asd_one"

	testEvent := Event{
		Id:            "1",
		Kind:          "ddl",
		Status:        "waiting_batch",
		TriggerTag:    "123",
		TriggerEvent:  "event",
		TransactionId: "456",
		Data:          &data,
	}

	event, err := db.GetEvent("1")

	if err != nil {
		t.Errorf("get event returned error: %v\n", err)
	}

	if !reflect.DeepEqual(event, testEvent) {
		t.Errorf(
			"get event => %#v, want %#v",
			event,
			testEvent,
		)
	}
}

func TestEventInsertQuery(t *testing.T) {
	db.db.Exec(`
		TRUNCATE teleport.event;
	`)

	data := "asd_one"

	testEvent := &Event{
		Id:            "5",
		Kind:          "ddl",
		Status:        "waiting_batch",
		TriggerTag:    "123",
		TriggerEvent:  "event",
		TransactionId: "456",
		Data:          &data,
	}

	tx := db.NewTransaction()
	testEvent.InsertQuery(tx)
	tx.Commit()

	event, _ := db.GetEvent("5")

	if !reflect.DeepEqual(event, *testEvent) {
		t.Errorf(
			"get event => %#v, want %#v",
			event,
			testEvent,
		)
	}
}

func TestEventUpdateQuery(t *testing.T) {
	db.db.Exec(`
		TRUNCATE teleport.event;
	`)

	data := "asd_one"

	testEvent := &Event{
		Id:            "5",
		Kind:          "ddl",
		Status:        "waiting_batch",
		TriggerTag:    "123",
		TriggerEvent:  "event",
		TransactionId: "456",
		Data:          &data,
	}

	tx := db.NewTransaction()
	testEvent.InsertQuery(tx)
	tx.Commit()

	tx = db.NewTransaction()
	testEvent.Status = "batched"
	testEvent.UpdateQuery(tx)
	tx.Commit()

	event, _ := db.GetEvent("5")

	if !reflect.DeepEqual(event, *testEvent) {
		t.Errorf(
			"get event => %#v, want %#v",
			event,
			testEvent,
		)
	}
}

func TestEventBelongsToBatch(t *testing.T) {
	db.db.Exec(`
		TRUNCATE teleport.event, teleport.batch_events;
	`)

	data := "asd_one"

	testEvent := &Event{
		Id:            "5",
		Kind:          "ddl",
		Status:        "waiting_batch",
		TriggerTag:    "123",
		TriggerEvent:  "event",
		TransactionId: "456",
		Data:          &data,
	}

	batch := &Batch{
		Id:     "2",
		Status: "waiting_transmission",
		Source: "asd",
		Target: "asd",
		Data:   nil,
	}

	tx := db.NewTransaction()
	testEvent.InsertQuery(tx)
	tx.Commit()

	tx = db.NewTransaction()
	testEvent.BelongsToBatch(tx, batch)
	tx.Commit()

	tx = db.NewTransaction()
	var batchId string
	tx.Get(&batchId, "SELECT batch_id FROM teleport.batch_events WHERE event_id = $1;", testEvent.Id)

	if batchId != batch.Id {
		t.Errorf("batch_id in batch_events table => %s, want %s", batchId, batch.Id)
	}
}

func TestEventSetDataFromAction(t *testing.T) {
	testEvent := &Event{
		Id:            "5",
		Kind:          "ddl",
		Status:        "waiting_batch",
		TriggerTag:    "123",
		TriggerEvent:  "event",
		TransactionId: "456",
		Data:          nil,
	}

	testEvent.SetDataFromAction(stubAction)

	if stubActionData != *testEvent.Data {
		t.Errorf("event data => %#v, want %#v", testEvent.Data, stubActionData)
	}
}

func TestEventGetActionFromData(t *testing.T) {
	testEvent := &Event{
		Id:            "5",
		Kind:          "ddl",
		Status:        "waiting_batch",
		TriggerTag:    "123",
		TriggerEvent:  "event",
		TransactionId: "456",
		Data:          nil,
	}

	testEvent.Data = &stubActionData

	action, err := testEvent.GetActionFromData()

	if err != nil {
		t.Errorf("get action from data returned error: %v", err)
	}

	if !reflect.DeepEqual(action, stubAction) {
		t.Errorf(
			"action data => %#v, want %#v",
			action,
			stubAction,
		)
	}
}
