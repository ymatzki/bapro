package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const (
	generation       = 3
	waitSecond       = 3
	snapshotBasePath = "/snapshots"
	suffix           = ".tar.gz"
)

type AwsConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Bucket          string
}

func main() {
	var isDaemon bool
	var rootCmd = &cobra.Command{
		Use:   "bapro [command]",
		Short: "Export/Import prometheus snapshot data to remote object storage.",
	}
	var saveCmd = &cobra.Command{
		Use:   "save [storage.tsdb.path]",
		Short: "Export prometheus snapshot data to remote object storage.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if isDaemon {
				autoSave(args[0] + snapshotBasePath)

				sigs := make(chan os.Signal, 1)
				done := make(chan bool, 1)
				go gracefulShutdown(sigs, done)
				log.Println("Start Bapro")
				<-done
				log.Println("End Bapro")
			} else {
				snapshotDataPath, err := getSnapshotDataPath(args[0] + snapshotBasePath)
				if err != nil {
					fmt.Errorf("file [%s] dose not exists", args[0]+snapshotBasePath)
					panic(err)
				}
				save(snapshotDataPath)
			}
		},
	}

	var loadCmd = &cobra.Command{
		Use:   "load",
		Short: "Import prometheus snapshot data to remote object storage.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			load(args[0])
		},
	}

	saveCmd.Flags().BoolVarP(
		&isDaemon,
		"daemon",
		"d",
		false,
		`Watch specified directory indefinitely. If snapshot directory is created, 
export to object storage.`,
	)
	rootCmd.AddCommand(saveCmd, loadCmd)
	rootCmd.Execute()
}

func gracefulShutdown(sigs chan os.Signal, done chan bool) {
	signal.Notify(sigs, syscall.SIGTERM)
	sig := <-sigs
	switch sig.String() {
	case syscall.SIGTERM.String():
		log.Println("graceful shutdown...")
		// TODO: decide appropriate sleep second
		time.Sleep(5 * time.Second)
	}
	log.Printf("Get signal: %s\n", sig.String())
	done <- true
}

func autoSave(path string) {
	for {
		snapshotDataPath, err := getSnapshotDataPath(path)
		if err != nil {
			continue
		}
		time.Sleep(time.Second * waitSecond)
		save(snapshotDataPath)
	}
}

func compress(dir string, file string) (err error) {
	baseDir := path.Dir(dir)
	zf, err := os.Create(strings.TrimLeft(file, baseDir))
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

		fineName := strings.TrimLeft(path, baseDir)

		header, err := tar.FileInfoHeader(info, fineName)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(fineName)

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
				log.Printf("create dir %s\n", co)
				// create directory
				err = os.MkdirAll(co, 0755)
				if err != nil {
					return err
				}
			}
		case tar.TypeReg:
			err = os.MkdirAll(path.Dir(co), 0755)
			if err != nil {
				return err
			}
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

func save(path string) {
	// TODO: Select object storage
	config := getAwsConfig()
	fileName := filepath.Base(path+suffix)
	compress(path, fileName)

	// Upload snapshot
	if err := upload(fileName, config); err != nil {
		log.Fatalf(err.Error())
	}

	clean(path)
	clean(fileName)
	// Delete old snapshot
	targets, err := list(config)
	if err != nil {
		log.Println(err)
		return
	}
	sortTargetsByTime(targets)
	deleteTargets := targets[generation:len(targets)]
	if len(deleteTargets) < 1 {
		log.Println("no delete targets")
		return
	}
	delete(deleteTargets, config)
}

func load(path string) {
	// TODO: Select object storage
	config := getAwsConfig()
	targets, err := list(config)
	if err != nil {
		log.Println(err)
		return
	}
	sortTargetsByTime(targets)
	getTarget := targets[0]
	download(*getTarget.Key, config)
	uncompress(*getTarget.Key, "./")
	install(strings.TrimRight(*getTarget.Key, suffix), path)
	clean(*getTarget.Key)
}

func getSnapshotDataPath(path string) (snapshotDataPath string, err error) {
	_, err = os.Stat(path)
	if err != nil {
		return "", err
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})

	if len(files) > 0 {
		return path + "/" + files[0].Name(), nil
	} else {
		return "", fmt.Errorf("file dose not exist.")
	}

}

func clean(path string) (err error) {
	return os.RemoveAll(path)
}

func install(snapshots string, dir string) (err error) {
	files, err := ioutil.ReadDir(snapshots)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.Rename(snapshots+"/"+file.Name(), dir+"/"+file.Name())
		if err != nil {
			return err
		}
	}
	return nil
}
