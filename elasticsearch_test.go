package errnie

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
startElasticTestServer returns an httptest server that accepts index requests.
*/
func startElasticTestServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("X-Elastic-Product", "Elasticsearch")
		response.Header().Set("Content-Type", "application/json")

		if request.Method == http.MethodGet && request.URL.Path == "/" {
			response.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(response, `{"version":{"number":"9.0.0"}}`)
			return
		}

		if request.Method != http.MethodPost {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !strings.Contains(request.URL.Path, "/_doc") {
			http.Error(response, "not found", http.StatusNotFound)
			return
		}

		response.WriteHeader(statusCode)
		_, _ = io.WriteString(response, body)
	}))
}

/*
TestNewElasticPostWriter verifies Elasticsearch sink construction and validation.
*/
func TestNewElasticPostWriter(t *testing.T) {
	Convey("Given a valid Elasticsearch URL and index", t, func() {
		server := startElasticTestServer(http.StatusCreated, `{"result":"created"}`)
		defer server.Close()

		Convey("When newElasticPostWriter is called", func() {
			sink, err := newElasticPostWriter(server.URL, "logs", "", "")

			Convey("Then it should return a configured sink", func() {
				So(err, ShouldBeNil)
				So(sink, ShouldNotBeNil)
				So(sink.index, ShouldEqual, "logs")
			})
		})
	})

	Convey("Given basic auth credentials", t, func() {
		server := startElasticTestServer(http.StatusCreated, `{"result":"created"}`)
		defer server.Close()

		Convey("When newElasticPostWriter is called with username and password", func() {
			sink, err := newElasticPostWriter(server.URL, "logs", "elastic", "secret")

			Convey("Then it should return a configured sink", func() {
				So(err, ShouldBeNil)
				So(sink, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a missing URL", t, func() {
		Convey("When newElasticPostWriter is called", func() {
			sink, err := newElasticPostWriter("  ", "logs", "", "")

			Convey("Then it should return a validation error", func() {
				So(sink, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "url and index required")
			})
		})
	})

	Convey("Given a missing index", t, func() {
		Convey("When newElasticPostWriter is called", func() {
			sink, err := newElasticPostWriter("http://localhost:9200", " ", "", "")

			Convey("Then it should return a validation error", func() {
				So(sink, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "url and index required")
			})
		})
	})
}

/*
TestElasticPostWriterWrite verifies indexing behaviour for the Elasticsearch sink.
*/
func TestElasticPostWriterWrite(t *testing.T) {
	Convey("Given an empty payload", t, func() {
		server := startElasticTestServer(http.StatusCreated, `{"result":"created"}`)
		defer server.Close()

		sink, err := newElasticPostWriter(server.URL, "logs", "", "")
		So(err, ShouldBeNil)

		Convey("When Write is called", func() {
			written, writeErr := sink.Write(nil)

			Convey("Then it should no-op successfully", func() {
				So(writeErr, ShouldBeNil)
				So(written, ShouldEqual, 0)
			})
		})
	})

	Convey("Given a valid JSON payload and a successful Elasticsearch response", t, func() {
		server := startElasticTestServer(http.StatusCreated, `{"result":"created"}`)
		defer server.Close()

		sink, err := newElasticPostWriter(server.URL, "logs", "", "")
		So(err, ShouldBeNil)
		payload := []byte(`{"level":"info","message":"hello"}`)

		Convey("When Write is called", func() {
			written, writeErr := sink.Write(payload)

			Convey("Then it should index the payload", func() {
				So(writeErr, ShouldBeNil)
				So(written, ShouldEqual, len(payload))
			})
		})
	})

	Convey("Given a valid payload and an Elasticsearch error response", t, func() {
		server := startElasticTestServer(http.StatusBadRequest, `{"error":"bad request"}`)
		defer server.Close()

		sink, err := newElasticPostWriter(server.URL, "logs", "", "")
		So(err, ShouldBeNil)
		payload := []byte(`{"level":"info","message":"hello"}`)

		Convey("When Write is called", func() {
			written, writeErr := sink.Write(payload)

			Convey("Then it should return an error", func() {
				So(written, ShouldEqual, 0)
				So(writeErr, ShouldNotBeNil)
				So(writeErr.Error(), ShouldContainSubstring, "elasticsearch:")
			})
		})
	})
}

var (
	benchmarkElasticSink      *elasticPostWriter
	benchmarkElasticWriteSink int
	benchmarkElasticWriteErr  error
	benchmarkElasticPayload   = []byte(`{"level":"info","message":"benchmark"}`)
)

/*
BenchmarkNewElasticPostWriter measures sink construction with and without auth.
*/
func BenchmarkNewElasticPostWriter(b *testing.B) {
	server := startElasticTestServer(http.StatusCreated, `{"result":"created"}`)
	defer server.Close()

	b.Run("without auth", func(b *testing.B) {
		for range b.N {
			benchmarkElasticSink, benchmarkElasticWriteErr = newElasticPostWriter(server.URL, "logs", "", "")
		}
	})

	b.Run("with auth", func(b *testing.B) {
		for range b.N {
			benchmarkElasticSink, benchmarkElasticWriteErr = newElasticPostWriter(server.URL, "logs", "elastic", "secret")
		}
	})
}

/*
BenchmarkElasticPostWriterWrite measures Write for empty and non-empty payloads.
*/
func BenchmarkElasticPostWriterWrite(b *testing.B) {
	server := startElasticTestServer(http.StatusCreated, `{"result":"created"}`)
	defer server.Close()

	sink, err := newElasticPostWriter(server.URL, "logs", "", "")
	if err != nil {
		b.Fatal(err)
	}

	b.Run("empty payload", func(b *testing.B) {
		for range b.N {
			benchmarkElasticWriteSink, benchmarkElasticWriteErr = sink.Write(nil)
		}
	})

	b.Run("json payload", func(b *testing.B) {
		for range b.N {
			benchmarkElasticWriteSink, benchmarkElasticWriteErr = sink.Write(benchmarkElasticPayload)
		}
	})
}
