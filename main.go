package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"
)

type AwsConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Bucket          string
}

func main() {
	awsConfig := &AwsConfig{
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_SESSION_TOKEN"),
		os.Getenv("AWS_DEFAULT_REGION"),
		os.Getenv("AWS_DEFAULT_BUCKET"),
	}
	//upload("test.txt", awsConfig)
	list(awsConfig)
}

func createCredentials(config *AwsConfig) (cred *credentials.Credentials) {
	cred = credentials.NewStaticCredentials(
		config.AccessKeyID,
		config.SecretAccessKey,
		config.SessionToken);
	return
}

func upload(filename string, config *AwsConfig) error {

	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: createCredentials(config),
		Region:      aws.String(config.Region),
	}))

	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filename, err)
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(filename),
		Body:   f,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	fmt.Printf("file uploaded to %s\n", aws.StringValue(&result.Location))
	return nil
}

func delete(filename string, config *AwsConfig) error {

	//sess := session.Must(session.NewSession(&aws.Config{
	//	Credentials: createCredentials(config),
	//	Region:      aws.String(config.Region),
	//}))

	//uploader := s3manager.NewUploader(sess)
	//
	//f, err := os.Open(filename)
	//if err != nil {
	//	return fmt.Errorf("failed to open file %q, %v", filename, err)
	//}
	//
	//result, err := uploader.Upload(&s3manager.UploadInput{
	//	Bucket: aws.String(config.Bucket),
	//	Key:    aws.String(filename),
	//	Body:   f,
	//})
	//
	//if err != nil {
	//	return fmt.Errorf("failed to upload file, %v", err)
	//}
	//
	//fmt.Printf("file uploaded to %s\n", aws.StringValue(&result.Location))
	return nil
}

func list(config *AwsConfig) error {

	svc := s3.New(session.New(&aws.Config{
		Credentials: createCredentials(config),
		Region:      aws.String(config.Region),
	}))

	input := &s3.ListObjectsInput{
		Bucket: aws.String(config.Bucket),
	}

	reuslt, err := svc.ListObjects(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return fmt.Errorf(s3.ErrCodeNoSuchBucket, aerr.Error())
			default:
				return fmt.Errorf(aerr.Error())
			}
		}
		return fmt.Errorf(err.Error())
	}

	fmt.Println(reuslt)

	return nil
}
