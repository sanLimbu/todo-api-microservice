package main

import (
	"embed"
	"flag"
	"fmt"
	"log"

	"github.com/sanLimbu/todo-api/cmd/internal"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
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

}
