package drivers

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitdlv/gokit/storage"
)

func NewLocal(root string) *Local {
	r, err := filepath.Abs(root)
	if err != nil {
		panic(err)
	}
	return &Local{
		root: r,
	}
}

type Local struct {
	root string
}

func (l *Local) ServeFile(w http.ResponseWriter, r *http.Request, path string) {
	content, err := l.GetBytes(path)
	if err != nil {
		println(err.Error())
		if errors.Is(err, storage.ErrFileNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
	w.Write(content)
}

func (l *Local) GetBytes(path string) (content []byte, err error) {
	fpath := l.filePath(path)
	f, err := os.Open(fpath)
	if err != nil {
		return
	}
	defer f.Close()
	content, err = io.ReadAll(f)
	return
}

func (l *Local) filePath(path string) string {
	return fmt.Sprintf(
		"%s/%s",
		strings.TrimRight(l.root, "/"),
		strings.TrimLeft(path, "/"),
	)
}

func (l *Local) PutBytes(content []byte, path string) (err error) {
	fpath := l.filePath(path)
	dir := filepath.Dir(fpath)
	err = os.MkdirAll(dir, fs.ModePerm)
	if err != nil {
		return
	}

	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = f.Write(content)
	if err != nil {
		return
	}
	return
}

func (l *Local) Has(path string) bool {
	_, err := os.Stat(l.filePath(path))
	if err != nil {
		return false
	} else {
		return true
	}
}
