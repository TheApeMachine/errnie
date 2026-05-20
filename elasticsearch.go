package errnie

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
)

/*
elasticPostWriter writes one JSON log line per Write call to Elasticsearch _doc API.
*/
type elasticPostWriter struct {
	client  *elasticsearch.Client
	index   string
	timeout time.Duration
}

func newElasticPostWriter(baseURL, index, username, password string) (*elasticPostWriter, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	idx := strings.TrimSpace(index)

	if base == "" || idx == "" {
		return nil, fmt.Errorf("elasticsearch: url and index required")
	}

	opts := []elasticsearch.Option{
		elasticsearch.WithAddresses(base),
	}

	if username != "" || password != "" {
		opts = append(opts, elasticsearch.WithBasicAuth(strings.TrimSpace(username), password))
	}

	client, err := elasticsearch.New(opts...)
	if err != nil {
		return nil, err
	}

	return &elasticPostWriter{
		client:  client,
		index:   idx,
		timeout: 15 * time.Second,
	}, nil
}

/*
Write.
*/
func (sink *elasticPostWriter) Write(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), sink.timeout)
	defer cancel()

	response, err := sink.client.Index(
		sink.index,
		bytes.NewReader(payload),
		sink.client.Index.WithContext(ctx),
	)
	if err != nil {
		return 0, err
	}

	defer response.Body.Close()

	if response.IsError() {
		return 0, fmt.Errorf("elasticsearch: %s", response.String())
	}

	return len(payload), nil
}
