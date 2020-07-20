package core

import (
	"encoding/base64"
	"errors"
	"net/http"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
)

// GetProxyResponseV2 converts the data passed to the response writer into
// an events.APIGatewayProxyResponse object.
// Returns a populated proxy response object. If the response is invalid, for example
// has no headers or an invalid status code returns an error.
func (r *ProxyResponseWriter) GetProxyResponseV2() (events.APIGatewayV2HTTPResponse, error) {
	r.notifyClosed()

	if r.status == defaultStatusCode {
		return events.APIGatewayV2HTTPResponse{}, errors.New("Status code not set on response")
	}

	var output string
	isBase64 := false

	bb := (&r.body).Bytes()

	if utf8.Valid(bb) {
		output = string(bb)
	} else {
		output = base64.StdEncoding.EncodeToString(bb)
		isBase64 = true
	}

	proxyHeaders := make(map[string]string)

	for h := range r.headers {
		proxyHeaders[h] = r.headers.Get(h)
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode:        r.status,
		Headers:           proxyHeaders,
		MultiValueHeaders: http.Header(r.headers),
		Body:              output,
		IsBase64Encoded:   isBase64,
	}, nil
}
