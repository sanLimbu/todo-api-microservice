package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sanLimbu/todo-api/cmd/internal"
	internaldomain "github.com/sanLimbu/todo-api/internal"
	"github.com/sanLimbu/todo-api/internal/elasticsearch"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
	"go.uber.org/zap"
)

func main() {

	var env string
	flag.StringVar(&env, "env", "", "Environment Variables filename")
	flag.Parse()

	errC, err := run(env)
	if err != nil {
		log.Fatalf("Couldn't run: %s", err)
	}

	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
	}
}

func run(env string) (<-chan error, error) {

	//Initialize the logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("zap.NewProduction %w", err)
	}

	//Load environment variables from the specified file
	if err := envvar.Load(env); err != nil {
		return nil, fmt.Errorf("envvar.Load %w", err)
	}

	//Initialize Vault provider
	vault, err := internal.NewVaultProvider()
	if err != nil {
		return nil, fmt.Errorf("internal.NewVaultProvider %w", err)
	}

	//Create configuration using Vault provider
	conf := envvar.New(vault)

	//Initialize ElasticSearch client
	es, err := internal.NewElasticSearch(conf)
	if err != nil {
		return nil, fmt.Errorf("internal.NewElasticSearch %w", err)
	}

	//Intialize the kafka consumer
	kafka, err := internal.NewKafkaConsumer(conf, "elasticsearch-indexer")
	if err != nil {
		return nil, fmt.Errorf("internal.NewKafkaConsumr %w", err)
	}

	//Initialize OpenTelemetry exporter
	if _, err = internal.NewOTExporter(conf); err != nil {
		return nil, fmt.Errorf("newOTExporter %w", err)
	}

	//Create the server instance
	srv := &Server{
		logger: logger,
		kafka:  kafka,
		task:   elasticsearch.NewTask(es),
		doneC:  make(chan struct{}),
		closeC: make(chan struct{}),
	}
	//Channel to receive errors
	errC := make(chan error, 1)

	//Create a context that listens for interrupt signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	//Goroutine to handle graceful shutdown
	go func() {
		<-ctx.Done()
		logger.Info("shutdown signal received")

		//Create a timeout context for shutdown
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		defer func() {
			//Cleanup resources
			logger.Sync()
			kafka.Consumer.Unsubscribe()
			stop()
			cancel()
			close(errC)
		}()

		//Attempt to gracefully shutdown the server
		if err := srv.Shutdown(ctxTimeout); err != nil {
			errC <- err
		}

		logger.Info("shutdown complete")

	}()

	//Goroutine to start the server
	go func() {
		logger.Info("Listening and serving")
		if err := srv.ListenAndServe(); err != nil {
			errC <- err
		}
	}()

	return errC, nil

}

//ListenAndServe
func (s *Server) ListenAndServe() error {
	// Helper function to commit message offsets
	commit := func(msg *kafka.Message) {
		if _, err := s.kafka.Consumer.CommitMessage(msg); err != nil {
			s.logger.Error("commit failed", zap.Error(err))
		}

	}

	// Start a Goroutine to handle Kafka message consumption
	go func() {
		run := true

		for run {
			select {
			case <-s.closeC: // Check if a signal to close has been received
				run = false
				break
			default: // Default case to poll messages
				msg, ok := s.kafka.Consumer.Poll(150).(*kafka.Message)
				if !ok {
					continue
				}

				// Decode the message value into an event struct
				var evt struct {
					Type  string
					Value internaldomain.Task
				}

				if err := json.NewDecoder(bytes.NewReader(msg.Value)).Decode(&evt); err != nil {
					s.logger.Info("Ignoring message, invalide", zap.Error(err))
					commit(msg)
					continue
				}

				ok = false

				// Handle the event based on its type
				switch evt.Type {
				case "tasks.event.updated", "tasks.event.created":
					if err := s.task.Index(context.Background(), evt.Value); err == nil {
						ok = true
					}

				case "tasks.event.deleted":
					if err := s.task.Delete(context.Background(), evt.Value.ID); err == nil {
						ok = true
					}
				}

				// Commit the message if processing was successful
				if ok {
					s.logger.Info("Consumed", zap.String("type", evt.Type))
					commit(msg)
				}
			}
		}

		// Log that the server is no longer processing messages and signal completion
		s.logger.Info("No more messages to consume, Exiting.")
		s.doneC <- struct{}{}

	}()
	return nil

}

type Server struct {
	logger *zap.Logger
	kafka  *internal.KafkaConsumer
	task   *elasticsearch.Task
	doneC  chan struct{}
	closeC chan struct{}
}

//Shutdown
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	close(s.closeC)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context.Done: %w", ctx.Err())

		case <-s.doneC:
			return nil
		}
	}
}
