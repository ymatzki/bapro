package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"
)

func main() {
	upload("test.txt")
}

func upload(filename string) error {

	cred := credentials.NewStaticCredentials("", "", "");
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: cred,
		Region:      aws.String("ap-northeast-1"),
	}))

	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", filename, err)
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("bapro"),
		Key:    aws.String(filename),
		Body:   f,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	fmt.Printf("fail uploaded to %s\n", aws.StringValue(&result.Location))
	return nil
}
