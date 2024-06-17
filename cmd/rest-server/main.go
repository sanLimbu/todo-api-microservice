package main

import (
	"context"
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

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	esv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	rv8 "github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riandyrn/otelchi"
	"github.com/sanLimbu/todo-api/cmd/internal"
	internaldomain "github.com/sanLimbu/todo-api/internal"
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
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "zap.newProduction")
	}

	if err := envvar.Load(env); err != nil {
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "envar.load")

	}

	vault, err := internal.NewVaultProvider()
	if err != nil {
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "internal.NewVaultProvider")
	}

	conf := envvar.New(vault)

	pool, err := internal.NewPostgreSQL(conf)
	if err != nil {
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "internal.NewPostgreSQl")

	}

	es, err := internal.NewElasticSearch(conf)
	if err != nil {
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "internal.NewElasticSearch")
	}

	kafka, err := internal.NewKafkaProducer(conf)
	if err != nil {
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "internal.NewKafkaProducer")
	}

	rdb, err := internal.NewRedis(conf)
	if err != nil {
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "internal.NewRedis")
	}

	promExporter, err := internal.NewOTExporter(conf)
	if err != nil {
		return nil, internaldomain.WrapErrorf(err, internaldomain.ErrorCodeUnkown, "internal.NewOTExporter")
	}

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
		DB:            pool,
		ElasticSearch: es,
		Kafka:         kafka,
		Metrics:       promExporter,
		Middlewares:   []func(next http.Handler) http.Handler{otelchi.Middleware("todo-api-server"), logging},
		Redis:         rdb,
		Logger:        logger,
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
			pool.Close()
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
	DB            *pgxpool.Pool
	ElasticSearch *esv7.Client
	Kafka         *internal.KafkaProducer
	RabbitMQ      *internal.RabbitMQ
	Redis         *rv8.Client
	Metrics       http.Handler
	Middlewares   []func(next http.Handler) http.Handler
	Logger        *zap.Logger
}

func newServer(conf serverConfig) (*http.Server, error) {
	router := chi.NewRouter()
	router.Use(render.SetContentType(render.ContentTypeJSON))
	for _, mw := range conf.Middlewares {
		router.Use(mw)
	}

	repo := postgresql.NewTask(conf.DB)
	search := elasticsearch.NewTask(conf.ElasticSearch)

	// msgBroker, err := rabbitmq.NewTask(conf.RabbitMQ.Channel)
	// if err != nil {
	// 	return nil, fmt.Errorf("rabbitmq.NewTask %w", err)
	// }

	msgBroker := kafka.NewTask(conf.Kafka.Producer, conf.Kafka.Topic)
	svc := service.NewTask(conf.Logger, repo, search, msgBroker)

	rest.RegisterOpenAPI(router)
	rest.NewTaskHandler(svc).Register(router)

	fsys, _ := fs.Sub(content, "static")
	router.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(fsys))))
	router.Handle("/metrics", conf.Metrics)

	lmt := tollbooth.NewLimiter(3, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Second})
	lmtmw := tollbooth.LimitHandler(lmt, router)

	return &http.Server{
		Handler:           lmtmw,
		Addr:              conf.Address,
		ReadTimeout:       1 * time.Second,
		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       1 * time.Second,
	}, nil
}
