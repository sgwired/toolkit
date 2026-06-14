package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Errorf("Expected string of length 10, got %d", len(s))
	}

}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed ", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer writer.Close()
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Errorf("%s: error creating form file: %v", e.name, err)
				return
			}
			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Errorf("%s: error opening test file: %v", e.name, err)
				return
			}
			defer f.Close()
			img, _, err := image.Decode(f)
			if err != nil {
				t.Errorf("%s: error decoding image: %v", e.name, err)
				return
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Errorf("%s: error encoding image: %v", e.name, err)
				return
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/upload", pr)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes
		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: unexpected error: %v", e.name, err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exists: %s, but it does not exist", e.name, err.Error())
			}
			// Clean up the uploaded file after the test
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer writer.Close()
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
			return
		}
		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Error(err)
			return
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
			return
		}
	}()

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/upload", pr)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exists: %s, but it does not exist", err.Error())
	}
	// Clean up the uploaded file after the test
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))

}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools

	testDir := "./testdata/newdir"
	err := testTools.CreateDirIfNotExist(testDir)
	if err != nil {
		t.Errorf("unexpected error creating directory: %v", err)
	}

	err = testTools.CreateDirIfNotExist(testDir)
	if err != nil {
		t.Errorf("unexpected error creating directory: %v", err)
	}

	_ = os.Remove(testDir)
}

var slugTest = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "normal string", s: "Now!! is the Time 123", expected: "now-is-the-time-123", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "string with only special characters", s: "!!!@@@###", expected: "", errorExpected: true},
	{name: "japanese characters", s: "こんにちは世界", expected: "", errorExpected: true},
	{name: "japanese characters and roman characters", s: "Hello worldこんにちは世界abc", expected: "hello-world-abc", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools
	for _, tt := range slugTest {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := testTools.Slugify(tt.s)
			if (err != nil) != tt.errorExpected {
				t.Errorf("%s: expected error: %v, got: %v", tt.name, tt.errorExpected, err)
			}
			if slug != tt.expected {
				t.Errorf("%s: expected slug: %s, got: %s", tt.name, tt.expected, slug)
			}
		})
	}
}
