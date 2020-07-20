package httpadapter

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
)

// ProxyV2 receives an API Gateway proxy event, transforms it into an http.Request
// object, and sends it to the http.HandlerFunc for routing.
// It returns a proxy response object generated from the http.Handler.
func (h *HandlerAdapter) ProxyV2(event events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	req, err := h.ProxyEventV2ToHTTPRequest(event)
	return h.proxyInternal(req, err)
}

// ProxyV2WithContext receives context and an API Gateway proxy event,
// transforms them into an http.Request object, and sends it to the http.Handler for routing.
// It returns a proxy response object generated from the http.ResponseWriter.
func (h *HandlerAdapter) ProxyV2WithContext(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	req, err := h.EventToRequestV2WithContext(ctx, event)
	return h.proxyInternalV2(req, err)
}

func (h *HandlerAdapter) proxyInternalV2(req *http.Request, err error) (events.APIGatewayV2HTTPResponse, error) {
	if err != nil {
		return core.GatewayTimeoutV2(), core.NewLoggedError("Could not convert proxy event to request: %v", err)
	}

	w := core.NewProxyResponseWriter()
	h.handler.ServeHTTP(http.ResponseWriter(w), req)

	resp, err := w.GetProxyResponseV2()
	if err != nil {
		return core.GatewayTimeoutV2(), core.NewLoggedError("Error while generating proxy response: %v", err)
	}

	return resp, nil
}
