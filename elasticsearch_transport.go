package errnie

import (
	"bytes"
	"io"
	"net/http"

	"github.com/valyala/fasthttp"
)

/*
fastHTTPTransport implements http.RoundTripper with valyala/fasthttp, following
the go-elasticsearch fasthttp example for lower-allocation HTTP requests.
*/
type fastHTTPTransport struct{}

/*
RoundTrip executes an HTTP request using fasthttp.
*/
func (transport *fastHTTPTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	fastRequest := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(fastRequest)

	fastResponse := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fastResponse)

	copyHTTPRequestToFastHTTP(fastRequest, request)

	if err := fasthttp.Do(fastRequest, fastResponse); err != nil {
		return nil, err
	}

	response := &http.Response{Header: make(http.Header)}
	copyFastHTTPResponseToHTTP(response, fastResponse)

	return response, nil
}

/*
copyHTTPRequestToFastHTTP converts a net/http request into a fasthttp request.
*/
func copyHTTPRequestToFastHTTP(destination *fasthttp.Request, source *http.Request) {
	method := source.Method
	if method == http.MethodGet && source.Body != nil {
		method = http.MethodPost
	}

	destination.SetHost(source.Host)
	destination.SetRequestURI(source.URL.String())
	destination.Header.SetMethod(method)

	for key, values := range source.Header {
		for _, value := range values {
			destination.Header.Set(key, value)
		}
	}

	if source.Body != nil {
		destination.SetBodyStream(source.Body, -1)
	}
}

/*
copyFastHTTPResponseToHTTP converts a fasthttp response into a net/http response.
The body is copied because the fasthttp response is released after RoundTrip.
*/
func copyFastHTTPResponseToHTTP(destination *http.Response, source *fasthttp.Response) {
	destination.StatusCode = source.StatusCode()

	source.Header.VisitAll(func(key, value []byte) {
		destination.Header.Set(string(key), string(value))
	})

	body := append([]byte(nil), source.Body()...)
	destination.Body = io.NopCloser(bytes.NewReader(body))
}
