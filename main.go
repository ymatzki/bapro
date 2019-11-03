package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
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
	//uncompress("/Users/ymatzki/Downloads/ccc.tar.gz", "/Users/ymatzki/Downloads/compress")
	load(getAwsConfig())

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

// TODO: implement


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
	config := getAwsConfig()
	for {
		time.Sleep(time.Second * waitSecond)
		_, err := os.Stat(path)
		if err == nil {
			// TODO: export snapshot
			save(config, path)
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

func save(config *AwsConfig, path string) {
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

func load(config *AwsConfig) {
	// TODO: Get targets from snapshot path
	targets, err := list(config)
	if err != nil {
		fmt.Println(err)
		return
	}
	sortTargetsByTime(targets)
	getTarget := targets[0]
	download(*getTarget.Key, config)
	uncompress(*getTarget.Key, "/Users/ymatzki/Downloads/compress")
}
