package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

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
