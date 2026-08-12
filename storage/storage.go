package storage

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

type Driver interface {
	PutBytes(content []byte, path string) (err error)
	GetBytes(path string) (content []byte, err error)
	Has(path string) bool
}

type DriverCanServeFile interface {
	ServeFile(w http.ResponseWriter, r *http.Request, path string)
}

// WithUrlPrefix 设置url前缀，设置后可拼接文件url
func WithUrlPrefix(urlPrefix string) func(storage *Storage) {
	return func(storage *Storage) {
		storage.urlPrefix = urlPrefix
	}
}

func NewStorage(driver Driver, opts ...func(*Storage)) *Storage {
	s := &Storage{
		drv: driver,
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Storage struct {
	drv       Driver
	urlPrefix string
}

// Put 保存文件
func (s *Storage) Put(content any, path string) (err error) {
	switch c := content.(type) {
	case []byte:
		err = s.drv.PutBytes(c, path)
	case string:
		err = s.drv.PutBytes([]byte(c), path)
	case multipart.FileHeader:
		var (
			f       multipart.File
			content []byte
		)
		f, err = c.Open()
		if err != nil {
			return
		}
		content, err = io.ReadAll(f)
		f.Close()
		if err != nil {
			return
		}
		err = s.drv.PutBytes(content, path)
	default:
		err = errors.New("not support type")
	}

	return
}

// ServeFile 将文件写道http.ResponseWriter，下载
func (s *Storage) ServeFile(w http.ResponseWriter, r *http.Request, path string) {
	if server, ok := s.drv.(DriverCanServeFile); ok {
		server.ServeFile(w, r, path)
	} else {
		panic(errors.New("can not serve file"))
	}
}

// Url 通过path获取url
func (s *Storage) Url(path string) string {
	if s.urlPrefix == "" {
		return ""
	}
	if !s.drv.Has(path) {
		return ""
	}
	return fmt.Sprintf(
		"%s/%s",
		strings.TrimRight(s.urlPrefix, "/"),
		strings.TrimLeft(path, "/"),
	)
}

// Has 是否存在path指定的文件
func (s *Storage) Has(path string) bool {
	return s.drv.Has(path)
}
