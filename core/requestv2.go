// Package core provides utility methods that help convert proxy events
// into an http.Request and http.ResponseWriter
package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

// GetAPIGatewayContextV2 extracts the API Gateway context object from a
// request's custom header.
// Returns a populated events.APIGatewayProxyRequestContext object from
// the request.
func (r *RequestAccessor) GetAPIGatewayContextV2(req *http.Request) (events.APIGatewayV2HTTPRequestContext, error) {
	if req.Header.Get(APIGwContextHeader) == "" {
		return events.APIGatewayV2HTTPRequestContext{}, errors.New("No context header in request")
	}
	context := events.APIGatewayV2HTTPRequestContext{}
	err := json.Unmarshal([]byte(req.Header.Get(APIGwContextHeader)), &context)
	if err != nil {
		log.Println("Erorr while unmarshalling context")
		log.Println(err)
		return events.APIGatewayV2HTTPRequestContext{}, err
	}
	return context, nil
}

// ProxyEventV2ToHTTPRequest converts an API Gateway proxy event into a http.Request object.
// Returns the populated http request with additional two custom headers for the stage variables and API Gateway context.
// To access these properties use the GetAPIGatewayStageVars and GetAPIGatewayContext method of the RequestAccessor object.
func (r *RequestAccessor) ProxyEventV2ToHTTPRequest(req events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	httpRequest, err := r.EventV2ToRequest(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return addToHeaderV2(httpRequest, req)
}

// EventToRequestV2WithContext converts an API Gateway proxy event and context into an http.Request object.
// Returns the populated http request with lambda context, stage variables and APIGatewayProxyRequestContext as part of its context.
// Access those using GetAPIGatewayContextFromContext, GetStageVarsFromContext and GetRuntimeContextFromContext functions in this package.
func (r *RequestAccessor) EventToRequestV2WithContext(ctx context.Context, req events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	httpRequest, err := r.EventV2ToRequest(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return addV2ToContext(ctx, httpRequest, req), nil
}

// EventV2ToRequest converts an API Gateway proxy event into an http.Request object.
// Returns the populated request maintaining headers
func (r *RequestAccessor) EventV2ToRequest(req events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	decodedBody := []byte(req.Body)
	if req.IsBase64Encoded {
		base64Body, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return nil, err
		}
		decodedBody = base64Body
	}

	path := req.RawPath
	if r.stripBasePath != "" && len(r.stripBasePath) > 1 {
		if strings.HasPrefix(path, r.stripBasePath) {
			path = strings.Replace(path, r.stripBasePath, "", 1)
		}
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	serverAddress := "https://" + req.RequestContext.DomainName
	if customAddress, ok := os.LookupEnv(CustomHostVariable); ok {
		serverAddress = customAddress
	}
	path = serverAddress + path

	if len(req.RawQueryString) > 0 {
		path += "?" + req.RawQueryString
	}

	httpMethod := req.RequestContext.HTTP.Method
	httpRequest, err := http.NewRequest(
		strings.ToUpper(httpMethod),
		path,
		bytes.NewReader(decodedBody),
	)

	if err != nil {
		fmt.Printf("Could not convert request %s:%s to http.Request\n", httpMethod, req.RawPath)
		log.Println(err)
		return nil, err
	}

	if req.Headers != nil {
		for k, v := range req.Headers {
			httpRequest.Header.Add(k, v)
		}
	}

	httpRequest.RequestURI = httpRequest.URL.RequestURI()

	return httpRequest, nil
}

func addToHeaderV2(req *http.Request, apiGwRequest events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	stageVars, err := json.Marshal(apiGwRequest.StageVariables)
	if err != nil {
		log.Println("Could not marshal stage variables for custom header")
		return nil, err
	}
	req.Header.Add(APIGwStageVarsHeader, string(stageVars))
	apiGwContext, err := json.Marshal(apiGwRequest.RequestContext)
	if err != nil {
		log.Println("Could not Marshal API GW context for custom header")
		return req, err
	}
	req.Header.Add(APIGwContextHeader, string(apiGwContext))
	return req, nil
}

func addV2ToContext(ctx context.Context, req *http.Request, apiGwRequest events.APIGatewayV2HTTPRequest) *http.Request {
	lc, _ := lambdacontext.FromContext(ctx)
	rc := requestContext{lambdaContext: lc, gatewayProxyContextV2: apiGwRequest.RequestContext, stageVars: apiGwRequest.StageVariables}
	ctx = context.WithValue(ctx, ctxKey{}, rc)
	return req.WithContext(ctx)
}

// GetAPIGatewayContextV2FromContext retrieve APIGatewayProxyRequestContext from context.Context
func GetAPIGatewayContextV2FromContext(ctx context.Context) (events.APIGatewayV2HTTPRequestContext, bool) {
	v, ok := ctx.Value(ctxKey{}).(requestContext)
	return v.gatewayProxyContextV2, ok
}
