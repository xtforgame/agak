// https://github.com/gorilla/websocket/blob/master/examples/echo/server.go
// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"archive/zip"
	"bytes"
	// "errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	uint16max = (1 << 16) - 1
	uint32max = (1 << 32) - 1
)

// https://golangcode.com/unzip-files-in-go/
func unzip(files []*zip.File, dest string) ([]string, error) {
	var filenames []string
	for _, f := range files {
		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return filenames, err
			}
		}
	}
	return filenames, nil
}

func Unzip(src string, dest string) ([]string, error) {
	var filenames []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()
	return unzip(r.File, dest)
}

func UnzipFromBytes(b []byte, f func(*zip.File) bool) (map[string][]byte, error) {
	filter := func(*zip.File) bool { return true }
	if f != nil {
		filter = f
	}
	resultMap := map[string][]byte{}
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return resultMap, err
	}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return resultMap, err
		}
		defer rc.Close()

		if !filter(f) {
			continue
		}
		if f.FileInfo().IsDir() {
			resultMap[f.Name] = nil
		} else {

			data, err := ioutil.ReadAll(rc)
			if err != nil {
				return resultMap, err
			}

			resultMap[f.Name] = data

			if err != nil {
				return resultMap, err
			}
		}
	}
	return resultMap, nil
}

func FileInfoHeader(
	name string,
	size int64,
	modTime time.Time,
	mode os.FileMode,
) (*zip.FileHeader, error) {
	fh := &zip.FileHeader{
		Name:               name,
		UncompressedSize64: uint64(size),
	}
	fh.SetModTime(modTime)
	fh.SetMode(mode)
	if fh.UncompressedSize64 > uint32max {
		fh.UncompressedSize = uint32max
	} else {
		fh.UncompressedSize = uint32(fh.UncompressedSize64)
	}
	fh.Method = zip.Deflate
	return fh, nil
}

type FileInBytes struct {
	Filename     string
	Bytes        []byte
	LastModified time.Time
	Mode         os.FileMode
}

type ZipFromBytesMapOptions struct {
	FolderPrefix string
	GetFilename  func(key string, fib *FileInBytes, options *ZipFromBytesMapOptions) string
}

func ZipFromBytesMap(
	bytesMap map[string]FileInBytes,
	options ZipFromBytesMapOptions,
) ([]byte, error) {
	fmt.Println("zip folder : (", options.FolderPrefix, "/", len(bytesMap), ")")

	buf := new(bytes.Buffer)
	// Create a new zip archive.
	zipWriter := zip.NewWriter(buf)
	for key, fib := range bytesMap {
		getFilename := func(key string, fib *FileInBytes, options *ZipFromBytesMapOptions) string {
			folderPath := ""
			if options.FolderPrefix != "" {
				folderPath = options.FolderPrefix
			}
			filename := key
			if fib.Filename != "" {
				filename = fib.Filename
			}
			return folderPath + "/" + filename
		}

		if options.GetFilename != nil {
			getFilename = options.GetFilename
		}

		filename := getFilename(key, &fib, &options)

		if filename == "" {
			continue
		}

		header, _ := FileInfoHeader(
			filename,
			int64(len(fib.Bytes)),
			fib.LastModified,
			fib.Mode,
		)
		zipFile, _ := zipWriter.CreateHeader(header)
		_, err := zipFile.Write(fib.Bytes)
		if err != nil {
			zipWriter.Close()
			return nil, err
		}
	}
	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}

	b := buf.Bytes()

	fmt.Println("zip folder : Done")

	// ===========

	// zipReader, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	// if err != nil {
	// 	return b, nil
	// }
	// for _, zipFile := range zipReader.File {
	// 	fmt.Println("Reading file:", zipFile.Name)
	// 	unzippedFileBytes, err := readZipFile(zipFile)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	_ = unzippedFileBytes // this is unzipped file bytes
	// }

	// ================
	return b, nil
}

// ==============

// https://stackoverflow.com/questions/37869793/how-do-i-zip-a-directory-containing-sub-directories-or-files-in-golang
func ZipLocalFolder(baseFolder string, baseInZip string) ([]byte, error) {
	// Create a new zip archive.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	zipWriter := zip.NewWriter(buf)

	// Add some files to the archive.
	err := addFiles(zipWriter, baseFolder, baseInZip)

	if err != nil {
		zipWriter.Close()
		return nil, err
	}

	// Make sure to check the error on Close.
	err = zipWriter.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addFiles(zipWriter *zip.Writer, basePath, baseInZip string) error {
	// Open the Directory
	fileInfos, err := ioutil.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileInfos {
		nextPath := filepath.Join(basePath, fileInfo.Name())
		nextPathInZip := filepath.Join(baseInZip, fileInfo.Name())

		if !fileInfo.IsDir() {
			dat, err := ioutil.ReadFile(nextPath)
			if err != nil {
				return err
			}

			header, _ := FileInfoHeader(
				nextPathInZip,
				fileInfo.Size(),
				fileInfo.ModTime(),
				fileInfo.Mode(),
			)
			zipFile, _ := zipWriter.CreateHeader(header)
			_, err = zipFile.Write(dat)
			if err != nil {
				return err
			}
		} else {
			// Recurse
			err := addFiles(zipWriter, nextPath, nextPathInZip)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
