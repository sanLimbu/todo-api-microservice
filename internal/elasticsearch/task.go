package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	esv7 "github.com/elastic/go-elasticsearch/v7"
	esv7api "github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/sanLimbu/todo-api/internal"
	"go.opentelemetry.io/otel/trace"
)

//Task represents the repository used for interacting with Task records
type Task struct {
	client *esv7.Client
	index  string
}

type indexedTask struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Priority    internal.Priority `json:"priority"`
	IsDone      bool              `json:"is_done"`
	DateStart   int64             `json:"date_start"`
	DateDue     int64             `json:"date_due"`
}

//NewTask instantiates the Task repository
func NewTask(client *esv7.Client) *Task {
	return &Task{
		client: client,
		index:  "tasks",
	}
}

//Index creates or updates a task in an index.
func (t *Task) Index(ctx context.Context, task internal.Task) error {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.index")
	defer span.End()

	body := indexedTask{
		ID:          task.ID,
		Description: task.Description,
		IsDone:      task.IsDone,
		DateStart:   task.Dates.Start.UnixNano(),
		DateDue:     task.Dates.Due.UnixNano(),
	}

	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "json.NewEncoder.Encode")

	}

	req := esv7api.IndexRequest{
		Index:      t.index,
		Body:       &buf,
		DocumentID: task.ID,
		Refresh:    "true",
	}

	resp, err := req.Do(ctx, t.client)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "IndexRequest.Do")
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return internal.NewErrorf(internal.ErrorCodeUnkown, "IndexRequest.Do %s", resp.StatusCode)
	}
	io.Copy(io.Discard, resp.Body)
	return nil

}

//Delete removes a task from the index
func (t *Task) Delete(ctx context.Context, id string) error {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Delete")
	defer span.End()

	req := esv7api.DeleteRequest{
		Index:      t.index,
		DocumentID: id,
	}

	resp, err := req.Do(ctx, t.client)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeNotFound, "DeleteRequest.Do")
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return internal.NewErrorf(internal.ErrorCodeUnkown, "DeleteRequest.Do %s", resp.StatusCode)
	}

	io.Copy(io.Discard, resp.Body)

	return nil

}

//Search returns tasks matching a query
func (t *Task) Search(ctx context.Context, description *string, priority *internal.Priority, isDone *bool) ([]internal.Task, error) {

	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Search")
	defer span.End()

	if description == nil && priority == nil && isDone == nil {
		return nil, nil
	}

	should := make([]interface{}, 0, 3)
	if description != nil {
		should = append(should, map[string]interface{}{
			"match": map[string]interface{}{
				"description": *description,
			},
		})
	}

	if priority != nil {
		should = append(should, map[string]interface{}{
			"match": map[string]interface{}{
				"priority": *priority,
			},
		})
	}

	if isDone != nil {
		should = append(should, map[string]interface{}{
			"match": map[string]interface{}{
				"is_done": *isDone,
			},
		})
	}

	var query map[string]interface{}

	if len(should) > 1 {
		query = map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"should": should,
				},
			},
		}
	} else {
		query = map[string]interface{}{
			"query": should[0],
		}
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "json.NewEncode.Encode")
	}

	req := esv7api.SearchRequest{
		Index: []string{t.index},
		Body:  &buf,
	}

	resp, err := req.Do(ctx, t.client)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "SearchRequest.Do")
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, internal.NewErrorf(internal.ErrorCodeUnkown, "SearchRequest.Do %d", resp.StatusCode)
	}

	var hits struct {
		Hits struct {
			Hits []struct {
				Source indexedTask `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		fmt.Println("Error here", err)
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "json.NewDecoder.Decode")
	}
	res := make([]internal.Task, len(hits.Hits.Hits))
	for i, hit := range hits.Hits.Hits {
		res[i].ID = hit.Source.ID
		res[i].Description = hit.Source.Description
		res[i].Priority = internal.Priority(hit.Source.Priority)
		res[i].Dates.Due = time.Unix(0, hit.Source.DateDue).UTC()
		res[i].Dates.Start = time.Unix(0, hit.Source.DateStart).UTC()
	}
	return res, nil

}
