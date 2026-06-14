package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. Any variable of this type
// will have access to all the methods with a *Tools receiver.
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

// RamdomString returns a string of random characters of length n, using randomStringSource as
// the source of for the string
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)
}

// UploadedFile is a struct used to save infromation about an uploaded file
type UploadedFile struct {
	NewFileName       string
	OringinalFileName string
	FileSize          int64
}

// UploadOneFile uploads a single file from an http request. The first parameter is the http request,
// the second is the directory to which the file should be uploaded, and the third is an optional boolean
// indicating whether the file should be renamed. If the file should be renamed it uses the RandomString method to generate a new
// file name, with the original file extension.
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}
	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}
	return files[0], nil
}

// UploadFiles uploads one or more files from an http request. The first parameter is the http request,
// the second is the directory to which the file should be uploaded, and the third is an optional boolean
// indicating whether the file should be renamed. If the file should be renamed it uses the RandomString method to generate a new
// file name, with the original file extension. The method returns a slice of UploadedFile structs, one for each file uploaded,
// and an error if there was a problem with the upload.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}
	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	for _, fHeaders := range r.MultipartForm.File {
		for _, fhdr := range fHeaders {
			uploadedFiles, err = func(uploadeFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				inFile, err := fhdr.Open()
				if err != nil {
					return nil, err
				}
				defer inFile.Close()
				buff := make([]byte, 512)
				_, err = inFile.Read(buff)
				if err != nil {
					return nil, err
				}
				// Check to see if the file type file is permitted
				allowed := false
				fileType := http.DetectContentType(buff)

				if len(t.AllowedFileTypes) > 0 {
					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, x) {
							allowed = true
						}
					}
				} else {
					allowed = true
				}
				if !allowed {
					return nil, errors.New("the uploaded file type is not permitted")
				}

				_, err = inFile.Seek(0, 0)
				if err != nil {
					return nil, err
				}
				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(12), filepath.Ext(fhdr.Filename))
				} else {
					uploadedFile.NewFileName = fhdr.Filename
				}
				uploadedFile.OringinalFileName = fhdr.Filename

				var outfile *os.File
				defer outfile.Close()
				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, inFile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}
				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}

	}
	return uploadedFiles, nil
}

// CreateDirIFNotExists creates a directory and all necessary parents if it does not already exist. If the directory already exists, no error is returned.
func (t *Tools) CreateDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
	}
	return nil
}
