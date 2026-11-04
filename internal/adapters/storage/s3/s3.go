package s3

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mangashelf/mangashelf/internal/ports"
)

type S3 struct {
	client *s3.Client
	bucket string
}

type S3Config struct {
	Endpoint       string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	ForcePathStyle bool
}

// readSeekCloser envolve um io.ReadCloser adicionando Seek (no-op se não suportado).
type readSeekCloser struct {
	io.ReadCloser
	offset int64
}

func (r *readSeekCloser) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = 0 + offset
	}
	return r.offset, nil
}

func New(cfg S3Config) (*S3, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	}

	if cfg.Endpoint != "" {
		opts = append(opts, config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           cfg.Endpoint,
				SigningRegion: cfg.Region,
			}, nil
		})))
	}

	cfgAWS, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfgAWS, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})

	return &S3{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return &readSeekCloser{ReadCloser: out.Body}, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3) Presign(ctx context.Context, key string, op ports.Op, ttl time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	switch op {
	case ports.OpPut:
		req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = ttl
		})
		if err != nil {
			return "", err
		}
		return req.URL, nil
	case ports.OpGet:
		req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = ttl
		})
		if err != nil {
			return "", err
		}
		return req.URL, nil
	default:
		return "", errUnsupportedOp
	}
}

func (s *S3) List(ctx context.Context, prefix string) ([]ports.ObjectInfo, error) {
	out, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}
	var result []ports.ObjectInfo
	for _, obj := range out.Contents {
		result = append(result, ports.ObjectInfo{
			Key:     aws.ToString(obj.Key),
			Size:    aws.ToInt64(obj.Size),
			ETag:    aws.ToString(obj.ETag),
			ModTime: aws.ToTime(obj.LastModified),
		})
	}
	return result, nil
}

var _ ports.Storage = (*S3)(nil)

type errUnsupportedOpType struct{}

func (e errUnsupportedOpType) Error() string { return "unsupported s3 operation" }

var errUnsupportedOp = errUnsupportedOpType{}
