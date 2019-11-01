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
	"sort"
)

func getAwsConfig() *AwsConfig {
	awsConfig := &AwsConfig{
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_SESSION_TOKEN"),
		os.Getenv("AWS_DEFAULT_REGION"),
		os.Getenv("AWS_DEFAULT_BUCKET"),
	}
	return awsConfig
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

func download(filename string, config *AwsConfig) error {
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: createCredentials(config),
		Region:      aws.String(config.Region),
	}))

	downloader := s3manager.NewDownloader(sess)

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filename, err)
	}

	result, err := downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(config.Bucket),
			Key:    aws.String(filename),
		})
	if err != nil {
		return fmt.Errorf("failed to download file, %v", err)
	}

	fmt.Printf("file downloaded, %d bytes\n", result)
	return nil
}

func delete(targets []*s3.Object, config *AwsConfig) error {

	sess, err := session.NewSession(&aws.Config{
		Credentials: createCredentials(config),
		Region:      aws.String(config.Region),
	})
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	svc := s3.New(sess)
	var o []*s3.ObjectIdentifier
	for _, v := range targets {
		o = append(o, &s3.ObjectIdentifier{Key: v.Key})
	}
	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(config.Bucket),
		Delete: &s3.Delete{
			Objects: o,
			Quiet:   aws.Bool(false),
		},
	}
	result, err := svc.DeleteObjects(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Errorf(aerr.Error())
			}
		}
		return fmt.Errorf(err.Error())
	}

	fmt.Println(result)
	return nil
}

func list(config *AwsConfig) (contents []*s3.Object, err error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: createCredentials(config),
		Region:      aws.String(config.Region),
	})
	if err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	svc := s3.New(sess)

	input := &s3.ListObjectsInput{
		Bucket: aws.String(config.Bucket),
	}

	result, err := svc.ListObjects(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return nil, fmt.Errorf(s3.ErrCodeNoSuchBucket, aerr.Error())
			default:
				return nil, fmt.Errorf(aerr.Error())
			}
		}
		return nil, fmt.Errorf(err.Error())
	}

	return result.Contents, nil
}

// Sort by time modified. Most recently modified first.
func sortTargetsByTime(contents []*s3.Object) {
	sort.Slice(contents[:], func(i, j int) bool {
		return contents[i].LastModified.Local().After(contents[j].LastModified.Local())
	})
}
