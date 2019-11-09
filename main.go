package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	generation       = 3
	waitSecond       = 5
	snapshotBasePath = "/snapshots"
	suffix           = ".tar.gz"
	workDir          = "./"
)

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
			snapshotDir := args[0] + snapshotBasePath

			if isDaemon {
				log.Printf("Start bapro\n")
				go autoSave(snapshotDir)

				sigs := make(chan os.Signal, 1)
				done := make(chan bool, 1)
				go gracefulShutdown(sigs, done)
				<-done
				log.Printf("End bapro\n")
			} else {
				snapshotDataPath, err := getSnapshotDataPath(snapshotDir)
				if err != nil {
					fmt.Errorf("file [%s] dose not exists", snapshotDir)
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

func save(path string) {
	// TODO: Select object storage
	config := getAwsEnv()
	fileName := filepath.Base(path + suffix)
	compress(path, fileName)

	// Upload snapshot to object storage
	if err := upload(fileName, config); err != nil {
		log.Fatalf(err.Error())
	}

	// Delete local unnecessary file
	clean(path)
	clean(fileName)

	// Delete old snapshot from object storage
	targets, err := list(config)
	if err != nil {
		log.Println(err)
		return
	}
	sortTargetsByTime(targets)
	if generation < len(targets) {
		deleteTargets := targets[generation:len(targets)]
		if len(deleteTargets) < 1 {
			log.Println("no delete targets")
			return
		}
		delete(deleteTargets, config)
	}
}

func load(path string) {
	// TODO: Select object storage
	config := getAwsEnv()

	// Download latest file from object storage
	targets, err := list(config)
	if err != nil {
		log.Println(err)
		return
	}

	if len(targets) > 0 {
		sortTargetsByTime(targets)
		getTarget := targets[0]
		download(*getTarget.Key, config)
		uncompress(*getTarget.Key, workDir)
		install(strings.TrimRight(*getTarget.Key, suffix), path)
		clean(*getTarget.Key)
	}
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
		return "", fmt.Errorf("file does not exist.")
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
