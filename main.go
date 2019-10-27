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
	"os/signal"
	"sort"
	"syscall"
	"time"
)

const (
	generation = 3
)

type AwsConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Bucket          string
}

func main() {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	go gracefulShutdown(sigs, done)
	go checkPath("/tmp/path")

	fmt.Println("Start Bapro")
	<-done

	/**
	* TODO: Delete comment out after implement function to export snapshot
	awsConfig := &AwsConfig{
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_SESSION_TOKEN"),
		os.Getenv("AWS_DEFAULT_REGION"),
		os.Getenv("AWS_DEFAULT_BUCKET"),
	}

	// Upload snapshot
	upload("1572011713.txt", awsConfig)

	// Delete old snapshot
	targets, err := list(awsConfig)
	if err != nil {
		fmt.Println(err)
		return
	}

	deleteTargets := listDeleteTargets(targets)

	if len(deleteTargets) < 1 {
		fmt.Println("no delete targets")
		return
	}
	delete(deleteTargets, awsConfig)
	*/

}

// TODO: implement function to export snapshot
// TODO: implement function import snapshot

func gracefulShutdown(sigs chan os.Signal, done chan bool) {
	signal.Notify(sigs, syscall.SIGTERM)
	sig := <-sigs
	switch sig.String() {
	case syscall.SIGTERM.String():
		// TODO: handle SIGTERM
		fmt.Println("graceful shutdown...")
		time.Sleep(5 * time.Second)
	}
	fmt.Printf("Get signal: %s\n", sig.String())
	done <- true
}

func checkPath(path string) {
	for {
		time.Sleep(time.Second)
		_, err := os.Stat(path)
		if err == nil {
			fmt.Println("Directory or file [%s] exits.", path)
		}
		if os.IsNotExist(err) {
			fmt.Println("Directory or file [%s] dose Not exits.", path)
		}
	}
}

//----  AWS S3  ----

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

func delete(targets []*s3.Object, config *AwsConfig) error {
	svc := s3.New(session.New(&aws.Config{
		Credentials: createCredentials(config),
		Region:      aws.String(config.Region),
	}))
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

func listDeleteTargets(contents []*s3.Object) (targets []*s3.Object) {
	sort.Slice(contents[:], func(i, j int) bool {
		return contents[i].LastModified.Local().After(contents[j].LastModified.Local())
	})

	return contents[generation:len(contents)]
}

func list(config *AwsConfig) (contents []*s3.Object, err error) {
	svc := s3.New(session.New(&aws.Config{
		Credentials: createCredentials(config),
		Region:      aws.String(config.Region),
	}))

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
