package s3

import (
	"context"
	"io"
	"time"

	"github.com/mangashelf/mangashelf/internal/ports"
)

type S3 struct{}

func New() *S3 { return &S3{} }

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64) error { return errNotImplemented }
func (s *S3) Get(ctx context.Context, key string) (io.ReadSeekCloser, error) { return nil, errNotImplemented }
func (s *S3) Delete(ctx context.Context, key string) error { return errNotImplemented }
func (s *S3) Presign(ctx context.Context, key string, op ports.Op, ttl time.Duration) (string, error) { return "", errNotImplemented }
func (s *S3) List(ctx context.Context, prefix string) ([]ports.ObjectInfo, error) { return nil, errNotImplemented }

var _ ports.Storage = (*S3)(nil)

var errNotImplemented = &errImpl{msg: "s3 adapter not yet implemented"}

type errImpl struct{ msg string }

func (e *errImpl) Error() string { return e.msg }
