package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/godatei/datei/internal/config"
)

type s3Store struct {
	config config.S3Config
	client *s3.Client
}

// GetObject implements [Store].
func (s *s3Store) GetObject(ctx context.Context, reference string) (io.ReadCloser, error) {
	o, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.config.Bucket,
		Key:    &reference,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}
	return o.Body, nil
}

// Initialize implements [Store].
func (s *s3Store) Initialize(ctx context.Context) error {
	if s.config.CreateBucket {
		slog.Debug("creating bucket")
		_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: &s.config.Bucket,
			CreateBucketConfiguration: &types.CreateBucketConfiguration{
				LocationConstraint: types.BucketLocationConstraint(s.config.Region),
			},
		})
		if err != nil {
			if _, ok := errors.AsType[*types.BucketAlreadyOwnedByYou](err); ok {
				slog.Info("bucket already exists")
			} else {
				return fmt.Errorf("create bucket: %w", err)
			}
		} else {
			slog.Info("bucket created")
		}
	}

	return nil
}

// PutObject implements [Store].
func (s *s3Store) PutObject(ctx context.Context, data io.Reader, contentType string) (string, int64, error) {
	var rs io.ReadSeeker
	if drs, ok := data.(io.ReadSeeker); ok {
		rs = drs
	} else {
		if buf, err := io.ReadAll(data); err != nil {
			return "", 0, fmt.Errorf("read data: %w", err)
		} else {
			rs = bytes.NewReader(buf)
		}
	}

	var size int64
	h := sha256.New()
	if s, err := io.Copy(h, rs); err != nil {
		return "", 0, fmt.Errorf("generate sha256: %w", err)
	} else {
		size = s
	}

	key := hex.EncodeToString(h.Sum(nil))

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return "", 0, fmt.Errorf("reset reader: %w", err)
	}

	ho, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.config.Bucket,
		Key:    &key,
	})
	if err == nil {
		slog.Debug("object already exists", "key", key)
		return key, *ho.ContentLength, nil
	} else if _, ok := errors.AsType[*types.NotFound](err); !ok {
		return "", 0, fmt.Errorf("s3 head object: %w", err)
	}

	slog.Debug("creating object", "key", key)

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.config.Bucket,
		Key:         &key,
		Body:        rs,
		ContentType: &contentType,
	})
	if err != nil {
		return "", 0, fmt.Errorf("s3 put object: %w", err)
	}

	return key, size, nil
}

// DeleteObject implements [Store].
func (s *s3Store) DeleteObject(ctx context.Context, reference string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.config.Bucket,
		Key:    &reference,
	})
	if err != nil {
		return fmt.Errorf("s3 delete object: %w", err)
	}
	return nil
}

func NewS3Store(ctx context.Context, cfg config.S3Config) Store {
	var s3Client *s3.Client
	if config, err := awsconfig.LoadDefaultConfig(ctx); err != nil {
		slog.Debug("aws default config not available", "error", err)
		s3Client = s3.New(s3.Options{}, clientOpts(cfg))
	} else {
		slog.Debug("using aws default config")
		s3Client = s3.NewFromConfig(config, clientOpts(cfg))
	}
	return &s3Store{client: s3Client, config: cfg}
}

func clientOpts(s3Config config.S3Config) func(o *s3.Options) {
	return func(o *s3.Options) {
		o.Region = s3Config.Region
		o.UsePathStyle = s3Config.UsePathStyle

		if s3Config.Endpoint != "" {
			o.BaseEndpoint = new(s3Config.Endpoint)
		}

		if s3Config.AccessKeyID != "" && s3Config.SecretAccessKey != "" {
			o.Credentials = aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(s3Config.AccessKeyID, s3Config.SecretAccessKey, ""),
			)
		}
	}
}
