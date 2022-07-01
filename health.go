package main

import (
	"encoding/json"

	"github.com/JKC-Project/smalldomains.forwarder/smalldomains"
	"github.com/aws/aws-lambda-go/events"
)

type healthCheckPayload struct {
	IsSmallDomainsClientHealthy bool `json:"canAccessSmallDomainsGetter"`
}

func (this healthCheckPayload) areAllHealthChecksOk() bool {
	return this.IsSmallDomainsClientHealthy
}

func constructHealthCheckResponse(client smalldomains.Client) (events.ALBTargetGroupResponse) {
	healthChecks := healthCheckPayload{
		IsSmallDomainsClientHealthy: client.IsHealthy(),
	}

	healthCheckResponseBodyBytes, parseError := json.MarshalIndent(healthChecks, "", "  ")
	healthCheckResponseBody := string(healthCheckResponseBodyBytes)

	if parseError == nil && healthChecks.areAllHealthChecksOk() {
		return events.ALBTargetGroupResponse{
			StatusCode:        200,
			StatusDescription: "Health Check OK.",
			Body:              healthCheckResponseBody,
		}
	} else {
		return events.ALBTargetGroupResponse{
			StatusCode:        503,
			StatusDescription: "Health Check Bad.",
			Body:              healthCheckResponseBody,
		}
	}
}