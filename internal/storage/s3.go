package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
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
func (s *s3Store) PutObject(
	ctx context.Context,
	data io.Reader,
	name, contentType string,
) (*PutObjectOutput, error) {
	var rs io.ReadSeeker
	if drs, ok := data.(io.ReadSeeker); ok {
		rs = drs
	} else {
		if buf, err := io.ReadAll(data); err != nil {
			return nil, fmt.Errorf("read data: %w", err)
		} else {
			rs = bytes.NewReader(buf)
		}
	}

	var size int64
	h := sha256.New()
	if s, err := io.Copy(h, rs); err != nil {
		return nil, fmt.Errorf("generate sha256: %w", err)
	} else {
		size = s
	}

	digest := h.Sum(nil)
	checksum := hex.EncodeToString(digest)
	checksumB64 := base64.StdEncoding.EncodeToString(digest)

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("reset reader: %w", err)
	}

	var s3Key string
	s3Prefix := path.Join("data", time.Now().UTC().Format("2006/01/02"))

	for {
		for i := 0; ; i++ {
			if i == 0 {
				s3Key = name
			} else {
				ext := path.Ext(name)
				s3Key = fmt.Sprintf("%v (%v)%v", strings.TrimSuffix(name, ext), i, ext)
			}

			s3Key = path.Join(s3Prefix, s3Key)

			_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket: &s.config.Bucket,
				Key:    &s3Key,
			})

			if _, isNotFound := errors.AsType[*types.NotFound](err); isNotFound {
				break
			} else if err != nil {
				return nil, fmt.Errorf("s3 head object: %w", err)
			} else {
				slog.Debug("object already exists", "key", s3Key)
			}
		}

		slog.Debug("creating object", "key", s3Key)

		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:         &s.config.Bucket,
			Key:            &s3Key,
			Body:           rs,
			ContentType:    &contentType,
			ChecksumSHA256: &checksumB64,
			// Prevent object overwrites
			// Reference: https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-writes.html
			IfNoneMatch: new("*"),
		})
		if err != nil {
			aerr, ok := errors.AsType[smithy.APIError](err)
			if ok && (aerr.ErrorCode() == "ConditionalRequestConflict" || aerr.ErrorCode() == "PreconditionFailed") {
				slog.Warn("conflict during s3 put object", "error", err)
				continue
			}
			return nil, fmt.Errorf("s3 put object: %w", err)
		} else {
			break
		}
	}

	return &PutObjectOutput{StorageKey: s3Key, Checksum: checksum, Size: size}, nil
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
