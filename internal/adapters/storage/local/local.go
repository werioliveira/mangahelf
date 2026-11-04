package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type Local struct {
	root string
}

func New(root string) *Local { return &Local{root: root} }

func (l *Local) path(key string) string { return filepath.Join(l.root, key) }

func (l *Local) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	p := l.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	return nil
}

func (l *Local) Get(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	return os.Open(l.path(key))
}

func (l *Local) Delete(ctx context.Context, key string) error {
	return os.Remove(l.path(key))
}

func (l *Local) Presign(ctx context.Context, key string, op ports.Op, ttl time.Duration) (string, error) {
	return "file://" + l.path(key), nil
}

func (l *Local) List(ctx context.Context, prefix string) ([]ports.ObjectInfo, error) {
	var out []ports.ObjectInfo
	err := filepath.Walk(l.root, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(l.root, p)
		if err != nil {
			return nil
		}
		if prefix == "" || len(rel) >= len(prefix) && rel[:len(prefix)] == prefix {
			out = append(out, ports.ObjectInfo{Key: rel, Size: fi.Size(), ModTime: fi.ModTime()})
		}
		return nil
	})
	return out, err
}

var _ ports.Storage = (*Local)(nil)

var errNotImplemented = errors.New("not implemented")
