package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/JKC-Project/smalldomains.forwarder/smalldomains"
	"github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

var envVars = getEnvVars()

func main() {
	lambda.Start(HandleLambdaEvent)
}

/**
ALB Response must set certain fields and their values must be in a certain format.
Otherwise, the ALB will ignore our Lambda's response and give off a 502 instead. See this article for more info
https://stackoverflow.com/questions/57562352/aws-alb-returning-502-from-lambda-instead-of-custom-http-status
**/
func HandleLambdaEvent(ctx context.Context, request events.ALBTargetGroupRequest) (resp events.ALBTargetGroupResponse, error error) {
	client, log := initialiseDependenciesForLambdaRequest(ctx)

	if request.Path == "/actuator/health" {
		log.Info("Doing health check...")
		return constructHealthCheckResponse(client), nil
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Internal Server Error: %+v", r)
			resp = constructInternalServerError()
		}
	}()

	if request.HTTPMethod != "GET" {
		log.Errorf("Request has an unacceptable HTTP method: %v", request.HTTPMethod)
		return constructMethodNotAllowedResponse(), nil
	}

	smallDomainAlias := extractSmallDomainAliasFromPath(request.Path)
	smallDomain, err := client.GetSmallDomain(smallDomainAlias)

	if err == nil {
		log.Infof("Successfully found a SmallDomain: %+v", smallDomain)
		return constructRedirectResponse(smallDomain.LargeDomain), nil
	} else {
		log.Errorf("Could not find a SmallDomain with alias: %v", smallDomainAlias)
		return constructNotFoundResponse(smallDomainAlias), nil
	}
}

func initialiseDependenciesForLambdaRequest(ctx context.Context) (client smalldomains.Client, log logrus.Entry) {
	lambdacontext, _ := lambdacontext.FromContext(ctx)

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	log = *logger.WithFields(logrus.Fields{
		"awsRequestId": lambdacontext.AwsRequestID,
	})
	log.Info("Logger initialised")

	client = smalldomains.Client{
		SmallDomainsGetterUrl: getEnvVars().SmallDomainsGetterUrl,
		Log:                   log,
	}
	log.Info("smallDomains.Client initialised")

	return
}

func extractSmallDomainAliasFromPath(path string) string {
	regex := regexp.MustCompile("([a-zA-Z0-9\\-_]+)$")
	return regex.FindString(path)
}

func constructRedirectResponse(url string) events.ALBTargetGroupResponse {
	return events.ALBTargetGroupResponse{
		StatusCode:        302,
		StatusDescription: "302 URL Shortner: Redirecting to aliased location.",
		Headers: map[string]string{
			"Location": url,
		},
	}
}

func constructNotFoundResponse(desiredSmallDomain string) events.ALBTargetGroupResponse {
	return events.ALBTargetGroupResponse{
		StatusCode:        404,
		StatusDescription: fmt.Sprintf("404 No SmallDomains found for %v", desiredSmallDomain),
		Body:              fmt.Sprintf("404 No SmallDomains found for %v", desiredSmallDomain),
		Headers:           map[string]string{},
	}
}

func constructMethodNotAllowedResponse() events.ALBTargetGroupResponse {
	return events.ALBTargetGroupResponse{
		StatusCode:        405,
		StatusDescription: "405 HTTP Method Not Allowed.",
		Headers: map[string]string{
			"Allow": "GET",
		},
	}
}

func constructInternalServerError() events.ALBTargetGroupResponse {
	return events.ALBTargetGroupResponse{
		StatusCode:        500,
		StatusDescription: "500 Internal Server Error.",
		Body:              "500 Internal Server Error.",
		Headers:           map[string]string{},
	}
}
