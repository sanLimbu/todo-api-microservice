package internal

import (
	"fmt"

	esv7 "github.com/elastic/go-elasticsearch/v7"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
)

//NewElasticSearch instantiates the ElasticSearch client using configuration dfined in environment variables.

func NewElasticSearch(conf *envvar.Configuration) (es *esv7.Client, err error) {
	es, err = esv7.NewDefaultClient()
	if err != nil {
		return nil, fmt.Errorf("elasticsearch.oepn %w", err)
	}

	res, err := es.Info()
	if err != nil {
		return nil, fmt.Errorf("es.Info %w", err)
	}

	defer func() {
		err = res.Body.Close()
	}()
	return es, nil

}
