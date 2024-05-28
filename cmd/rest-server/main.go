package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	esv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"

	"github.com/sanLimbu/todo-api/cmd/internal"
	"github.com/sanLimbu/todo-api/internal/elasticsearch"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
	"github.com/sanLimbu/todo-api/internal/kafka"
	"github.com/sanLimbu/todo-api/internal/postgresql"
	"github.com/sanLimbu/todo-api/internal/rest"
	"github.com/sanLimbu/todo-api/internal/service"
	"go.uber.org/zap"
)

var content embed.FS

func main() {
	var env, address string

	flag.StringVar(&env, "env", "", "Environment Variables filename")
	flag.StringVar(&address, "address", ":9234", "HTTP Server Address")
	flag.Parse()

	errC, err := run(env, address)
	if err != nil {
		log.Fatalf("Couldn't run: %s", err)
	}

	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
	}

}

func run(env, address string) (<-chan error, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("zap.newProduction %w", err)
	}

	if err := envvar.Load(env); err != nil {
		return nil, fmt.Errorf("envar.load %w", err)
	}

	vault, err := internal.NewVaultProvider()
	if err != nil {
		return nil, fmt.Errorf("internal.NewVaultProvider %w", &err)
	}

	conf := envvar.New(vault)

	db, err := internal.NewPostgreSQL(conf)
	if err != nil {
		return nil, fmt.Errorf("internal.NewPostgreSQL %w", err)
	}

	es, err := internal.NewElasticSearch(conf)
	if err != nil {
		return nil, fmt.Errorf("internal.NewElasticSearch %w", err)
	}

	kafka, err := internal.NewKafkaProducer(conf)
	if err != nil {
		return nil, fmt.Errorf("internal.NewKafkaProducer %w", err)
	}

	// promExporter, err := internal.NewOTExporter(conf)
	// if err != nil {
	// 	return nil, fmt.Errorf("internal.NewOTExporter %w", err)
	// }

	logging := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info(r.Method,
				zap.Time("time", time.Now()),
				zap.String("url", r.URL.String()),
			)
			h.ServeHTTP(w, r)
		})
	}

	srv, err := newServer(serverConfig{
		Address:       address,
		DB:            db,
		ElasticSearch: es,
		Kafka:         kafka,
		//Metrics: promExporter,
		Middlewares: []mux.MiddlewareFunc{otelmux.Middleware("todo-api-server"), logging},
	})

	if err != nil {
		return nil, fmt.Errorf("newServer %w", err)
	}

	errC := make(chan error, 1)

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {

		<-ctx.Done()
		logger.Info("Shutdown signal received")

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer func() {
			logger.Sync()
			db.Close()
			stop()
			cancel()
			close(errC)
		}()

		srv.SetKeepAlivesEnabled(false)

		if err := srv.Shutdown(ctxTimeout); err != nil {
			errC <- err
		}
		logger.Info("shutdown completed")

	}()

	go func() {
		logger.Info("Listening and serving", zap.String("address", address))
		// "ListenAndServe always returns a non-nil error. After Shutdown or Close, the returned error is
		// ErrServerClosed."

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errC <- err
		}

	}()

	return errC, nil

}

type serverConfig struct {
	Address       string
	DB            *sql.DB
	ElasticSearch *esv7.Client
	Kafka         *internal.KafkaProducer
	//RabbitMQ *internal.RabbitMQ
	Metrics     http.Handler
	Middlewares []mux.MiddlewareFunc
}

func newServer(conf serverConfig) (*http.Server, error) {
	r := mux.NewRouter()
	for _, mw := range conf.Middlewares {
		r.Use(mw)
	}

	repo := postgresql.NewTask(conf.DB)
	search := elasticsearch.NewTask(conf.ElasticSearch)

	// msgBroker, err := rabbitmq.NewTask(conf.RabbitMQ.Channel)
	// if err != nil {
	// 	return nil, fmt.Errorf("rabbitmq.NewTask %w", err)
	// }

	msgBroker := kafka.NewTask(conf.Kafka.Producer, conf.Kafka.Topic)
	svc := service.NewTask(repo, search, msgBroker)

	rest.RegisterOpenAPI(r)
	rest.NewTaskHandler(svc).Register(r)

	fsys, _ := fs.Sub(content, "static")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(fsys))))
	r.Handle("/metrics", conf.Metrics)

	return &http.Server{
		Handler:           r,
		Addr:              conf.Address,
		ReadTimeout:       1 * time.Second,
		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       1 * time.Second,
	}, nil
}
