package internal

import (
	esv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/sanLimbu/todo-api/internal"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
)

//NewElasticSearch instantiates the ElasticSearch client using configuration dfined in environment variables.

func NewElasticSearch(_ *envvar.Configuration) (es *esv7.Client, err error) {
	es, err = esv7.NewDefaultClient()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "elasticsearch.Open")
	}

	res, err := es.Info()
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "es.Info")
	}

	defer func() {
		err = res.Body.Close()
	}()
	return es, nil

}
