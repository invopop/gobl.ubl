package utog

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/gobl"
)

// newDocumentFrom creates a UBL Document from a GOBL file in the `test/data` folder
func newDocumentFrom(name string) (*gobl.Envelope, error) {
	src, err := os.Open(filepath.Join(getTestDataPath(), name))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	inData, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	return Convert(inData)
}

func getTestDataPath() string {
	return filepath.Join(getRootFolder(), "test", "data", "utog")
}

func getRootFolder() string {
	cwd, _ := os.Getwd()

	for !isRootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}
	return cwd
}

func isRootFolder(dir string) bool {
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if file.Name() == "go.mod" {
			return true
		}
	}

	return false
}

func removeLastEntry(dir string) string {
	lastEntry := "/" + filepath.Base(dir)
	i := strings.LastIndex(dir, lastEntry)
	return dir[:i]
}
