# SpartaCustomResource
Sparta-based application that includes a [Lambda-backed CustomResource](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources-lambda.html)

1. [Install Go](https://golang.org/doc/install)
1. `go get github.com/mweagle/SpartaCustomResource`
1. `cd ./SpartaCustomResource`
1. `go run main.go provision --s3Bucket YOUR_S3_BUCKET`
1. Visit the CloudWatch Logs view and confirm the CustomAction has executed during the `provision` step.
