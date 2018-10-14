package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "net/http/pprof" // include pprop
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	sparta "github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	spartaAWSResource "github.com/mweagle/Sparta/aws/cloudformation/resources"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////
// Lambda Function
////////////////////////////////////////////////////////////////////////////////
func helloWorld(ctx context.Context) (string, error) {
	logger, loggerOk := ctx.Value(sparta.ContextKeyLogger).(*logrus.Logger)
	if loggerOk {
		logger.Info("Accessing structured logger 🙌")
	}
	contextLogger, contextLoggerOk := ctx.Value(sparta.ContextKeyRequestLogger).(*logrus.Entry)
	if contextLoggerOk {
		contextLogger.Info("Accessing request-scoped log, with request ID field")
	} else if loggerOk {
		logger.Warn("Failed to access scoped logger")
	} else {
		fmt.Printf("Failed to access any logger")
	}
	return "Hello World 👋. Welcome to AWS Lambda! 🙌🎉🍾", nil
}

////////////////////////////////////////////////////////////////////////////////
//
// CloudFormation Custom Resource
//
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// 1 - Define the custom type
const spartaHelloWorldResourceType = "Custom::sparta::HelloWorldResource"

////////////////////////////////////////////////////////////////////////////////
// 2 - Define the request body

// SpartaCustomResourceRequest is what the UserProperties
// should be set to in the CustomResource invocation
type SpartaCustomResourceRequest struct {
	Message *gocf.StringExpr
}

////////////////////////////////////////////////////////////////////////////////
// 3 - Create an instance of the Command handler with the request body as
// an embedded struct

// SpartaHelloWorldResource is a simple POC showing how to create custom resources
type SpartaHelloWorldResource struct {
	gocf.CloudFormationCustomResource
	SpartaCustomResourceRequest
}

// Create implements resource create
func (command SpartaHelloWorldResource) Create(awsSession *session.Session,
	event *spartaAWSResource.CloudFormationLambdaEvent,
	logger *logrus.Logger) (map[string]interface{}, error) {

	requestPropsErr := json.Unmarshal(event.ResourceProperties, &command)
	if requestPropsErr != nil {
		return nil, requestPropsErr
	}
	logger.Info("create: ", command.Message.Literal)
	return map[string]interface{}{
		"Resource": "Created message: " + command.Message.Literal,
	}, nil
}

// Update implements resource update
func (command SpartaHelloWorldResource) Update(awsSession *session.Session,
	event *spartaAWSResource.CloudFormationLambdaEvent,
	logger *logrus.Logger) (map[string]interface{}, error) {
	requestPropsErr := json.Unmarshal(event.ResourceProperties, &command)
	if requestPropsErr != nil {
		return nil, requestPropsErr
	}

	logger.Info("update: ", command.Message.Literal)
	return nil, nil
}

// Delete implements resource delete
func (command SpartaHelloWorldResource) Delete(awsSession *session.Session,
	event *spartaAWSResource.CloudFormationLambdaEvent,
	logger *logrus.Logger) (map[string]interface{}, error) {
	requestPropsErr := json.Unmarshal(event.ResourceProperties, &command)
	if requestPropsErr != nil {
		return nil, requestPropsErr
	}
	logger.Info("delete: ", command.Message.Literal)
	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////
// 4 - Register the CloudFormation custom type provider
func init() {
	customResourceFactory := func(resourceType string) gocf.ResourceProperties {
		switch resourceType {
		case spartaHelloWorldResourceType:
			return &SpartaHelloWorldResource{}
		}
		return nil
	}
	gocf.RegisterCustomResourceProvider(customResourceFactory)
}

////////////////////////////////////////////////////////////////////////////////
// 5 - Hook it up
func customResourceHooks() *sparta.WorkflowHooks {
	// Add the custom resource decorator
	customResourceDecorator := func(context map[string]interface{},
		serviceName string,
		template *gocf.Template,
		S3Bucket string,
		S3Key string,
		buildID string,
		awsSession *session.Session,
		noop bool,
		logger *logrus.Logger) error {

		// 1. Ensure the Lambda Function is registered
		customResourceName, customResourceNameErr := sparta.EnsureCustomResourceHandler(serviceName,
			spartaHelloWorldResourceType,
			nil, // This custom action doesn't need to access other AWS resources
			[]string{},
			template,
			S3Bucket,
			S3Key,
			logger)

		if customResourceNameErr != nil {
			return customResourceNameErr
		}

		// 2. Create the request for the invocation of the lambda resource with
		// parameters
		spartaCustomResource := &SpartaHelloWorldResource{}
		spartaCustomResource.ServiceToken = gocf.GetAtt(customResourceName, "Arn")
		spartaCustomResource.Message = gocf.String("Custom resource activated!")

		resourceInvokerName := sparta.CloudFormationResourceName("SpartaCustomResource",
			fmt.Sprintf("%v", S3Bucket),
			fmt.Sprintf("%v", S3Key))

		// Add it
		template.AddResource(resourceInvokerName, spartaCustomResource)
		return nil
	}
	// Add the decorator to the template
	hooks := &sparta.WorkflowHooks{}
	hooks.ServiceDecorators = []sparta.ServiceDecoratorHookHandler{
		sparta.ServiceDecoratorHookFunc(customResourceDecorator),
	}
	return hooks
}

////////////////////////////////////////////////////////////////////////////////
// Main
func main() {
	lambdaFn := sparta.HandleAWSLambda("Hello World",
		helloWorld,
		sparta.IAMRoleDefinition{})

	sess := session.Must(session.NewSession())
	awsName, awsNameErr := spartaCF.UserAccountScopedStackName("MyCustomResourceStack",
		sess)
	if awsNameErr != nil {
		fmt.Print("Failed to create stack name\n")
		os.Exit(1)
	}

	// Create the lambda functions
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Setup the CustomResource WorkflowHooks to annotate
	// the template with the custom resource invocation
	hooks := customResourceHooks()

	err := sparta.MainEx(awsName,
		"Simple Sparta App that uses a Lambda Custom Resource",
		lambdaFunctions,
		nil,
		nil,
		hooks,
		false)
	if err != nil {
		os.Exit(1)
	}
}
