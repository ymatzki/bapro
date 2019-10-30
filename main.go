package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

const (
	generation = 3
	waitSecond = 3
)

type AwsConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Bucket          string
}

func main() {
	uncompress("/Users/ymatzki/Downloads/ccc.tar.gz", "/Users/ymatzki/Downloads/compress")

	// TODO: Delete comment out after implemented
	//sigs := make(chan os.Signal, 1)
	//done := make(chan bool, 1)
	//
	//go gracefulShutdown(sigs, done)
	//go checkPath("/tmp/path")
	//
	//fmt.Println("Start Bapro")
	//<-done
}

// TODO: implement function import snapshot

func gracefulShutdown(sigs chan os.Signal, done chan bool) {
	signal.Notify(sigs, syscall.SIGTERM)
	sig := <-sigs
	switch sig.String() {
	case syscall.SIGTERM.String():
		fmt.Println("graceful shutdown...")
		// TODO: decide appropriate sleep second
		time.Sleep(5 * time.Second)
	}
	fmt.Printf("Get signal: %s\n", sig.String())
	done <- true
}

func checkPath(path string) {
	for {
		time.Sleep(time.Second * waitSecond)
		_, err := os.Stat(path)
		if err == nil {
			// TODO: export snapshot
			handleUpdate(getAwsConfig(), path)
			fmt.Printf("Directory or file [%s] exits.\n", path)
		}
	}
}

func compress(dir string, file string) (err error) {
	zf, err := os.Create(file)
	if err != nil {
		return err
	}
	defer zf.Close()

	gz := gzip.NewWriter(zf)
	defer gz.Close()

	ta := tar.NewWriter(gz)
	defer ta.Close()

	info, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not directory.", dir)
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}

		header.Name = filepath.Base(path)

		if err := ta.WriteHeader(header); err != nil {
			return err
		}

		co, err := os.Open(path)
		if err != nil {
			return err
		}
		defer co.Close()

		_, err = io.Copy(ta, co)

		return err
	})
	return
}

func uncompress(file string, dir string) (err error) {
	// TODO: implement
	co, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, os.FileMode(600))
	if err != nil {
		return err
	}

	gz, err := gzip.NewReader(co)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gz)

	info, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not directory.", dir)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		co := filepath.Join(dir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			_, err = os.Stat(co)
			if os.IsNotExist(err) {
				// create directory
				err = os.MkdirAll(co, 0755)
				if err != nil {
					return err
				}
			}
		case tar.TypeReg:
			w, err := os.OpenFile(co, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(w, tr)
			if err != nil {
				return err
			}
			defer w.Close()
		}
	}
	return nil
}

func handleUpdate(config *AwsConfig, path string) {
	// TODO: Get targets from snapshot path
	compress("/tmp/prometheus/snapshot/3235", "/Users/ymatzki/Downloads/ccc.tar.gz")
	// Upload snapshot
	upload("1572011713.txt", config)
	// Delete old snapshot
	targets, err := list(config)
	if err != nil {
		fmt.Println(err)
		return
	}
	sortTargetsByTime(targets)
	deleteTargets := targets[generation:len(targets)]
	if len(deleteTargets) < 1 {
		fmt.Println("no delete targets")
		return
	}
	delete(deleteTargets, config)
}

//----  AWS S3  ----

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

func download(targets []*s3.Object, config *AwsConfig) {

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
