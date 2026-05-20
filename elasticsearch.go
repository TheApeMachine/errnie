package errnie

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
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

var elasticPayloadReaderPool sync.Pool

/*
borrowElasticPayloadReader returns a pooled bytes.Reader positioned at payload.
*/
func borrowElasticPayloadReader(payload []byte) *bytes.Reader {
	reader, ok := elasticPayloadReaderPool.Get().(*bytes.Reader)
	if !ok {
		return bytes.NewReader(payload)
	}

	reader.Reset(payload)

	return reader
}

/*
releaseElasticPayloadReader returns a bytes.Reader to the pool after the request
body has been consumed by the client.
*/
func releaseElasticPayloadReader(reader *bytes.Reader) {
	if reader == nil {
		return
	}

	elasticPayloadReaderPool.Put(reader)
}

/*
elasticClientOptions builds client options tuned for high-volume log shipping:
fasthttp transport, no retries on failure, and automatic response draining for
connection reuse.
*/
func elasticClientOptions(baseURL, username, password string) []elasticsearch.Option {
	opts := []elasticsearch.Option{
		elasticsearch.WithAddresses(baseURL),
		elasticsearch.WithAutoDrainBody(),
		elasticsearch.WithDisableMetaHeader(),
		elasticsearch.WithTransportOptions(
			elastictransport.WithDisableRetry(),
			elastictransport.WithTransport(&fastHTTPTransport{}),
		),
	}

	if username != "" || password != "" {
		opts = append(opts, elasticsearch.WithBasicAuth(strings.TrimSpace(username), password))
	}

	return opts
}

/*
newElasticPostWriter builds an Elasticsearch log sink using the official
go-elasticsearch client with a fasthttp-backed transport. baseURL and index are
required; username and password enable HTTP basic auth when either is non-empty.
*/
func newElasticPostWriter(baseURL, index, username, password string) (*elasticPostWriter, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	idx := strings.TrimSpace(index)

	if base == "" || idx == "" {
		return nil, fmt.Errorf("elasticsearch: url and index required")
	}

	client, err := elasticsearch.New(elasticClientOptions(base, username, password)...)
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
Write indexes one JSON log line into Elasticsearch via the _doc API. Empty
payloads are ignored. Non-success HTTP responses are returned as errors.
*/
func (sink *elasticPostWriter) Write(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), sink.timeout)
	defer cancel()

	reader := borrowElasticPayloadReader(payload)

	response, err := sink.client.Index(
		sink.index,
		reader,
		sink.client.Index.WithContext(ctx),
	)

	releaseElasticPayloadReader(reader)

	if err != nil {
		return 0, err
	}

	defer response.Body.Close()

	if response.IsError() {
		return 0, fmt.Errorf("elasticsearch: %s", response.Status())
	}

	return len(payload), nil
}
