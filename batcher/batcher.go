package batcher

import (
	"github.com/pagarme/teleport/client"
	"github.com/pagarme/teleport/batcher/ddldiff"
	"github.com/pagarme/teleport/database"
	"log"
	"time"
)

type Batcher struct {
	db      *database.Database
	targets map[string]*client.Client
}

func New(db *database.Database, targets map[string]*client.Client) *Batcher {
	return &Batcher{
		db:      db,
		targets: targets,
	}
}

// Every sleepTime interval, create a batch with unbatched events
func (b *Batcher) Watch(sleepTime time.Duration) {
	for {
		err := b.createBatches()

		if err != nil {
			log.Printf("Error creating batch! %v\n", err)
		}

		time.Sleep(sleepTime)
	}
}

// Group all events 'waiting_batch' and create a batch with them.
func (b *Batcher) createBatches() error {
	// Get events waiting replication
	events, err := b.db.GetEvents("waiting_batch")

	if err != nil {
		return err
	}

	// Stop if there are no events
	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		b.processEvent(event)
	}

	// for targetName, _ := range b.targets {
	// 	// Start a transaction
	// 	tx := b.db.NewTransaction()
	//
	// 	// Allocate a new batch
	// 	batch := database.NewBatch()
	//
	// 	// Set events
	// 	batch.SetEvents(events)
	//
	// 	// Mark events as batched
	// 	for _, event := range events {
	// 		event.Status = "batched"
	// 		event.UpdateQuery(tx)
	// 	}
	//
	// 	// Set source and target
	// 	batch.Source = b.db.Name
	// 	batch.Target = targetName
	//
	// 	// Insert batch
	// 	batch.InsertQuery(tx)
	//
	// 	// Commit to database, returning errors
	// 	err := tx.Commit()
	//
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	log.Printf("Generated new batch: %v\n", batch)
	// }

	return nil
}

func (b *Batcher) processEvent(event database.Event) {
	if event.Kind == "ddl" {
		log.Printf("processing ddl change! %v\n", event.Id)
		diff := ddldiff.New(event)
		log.Printf("diff.PreSchemas: %v\n", diff.PreSchemas)
		log.Printf("diff.PostSchemas: %v\n", diff.PostSchemas)
	} else if event.Kind == "dml" {
		// Implement DML processor
	}
}