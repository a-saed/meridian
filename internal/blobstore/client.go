package blobstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client performs minimal S3-compatible Get/Put using SigV4.
type Client struct {
	http *http.Client
	cfg  Config
}

// NewClient builds an HTTP client for the given config. cfg must pass Validate when Enabled.
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if !cfg.Enabled() {
		return nil, fmt.Errorf("blobstore: NewClient called with disabled config")
	}
	return &Client{http: &http.Client{Timeout: 120 * time.Second}, cfg: cfg}, nil
}

// GetS3Object implements file.S3ObjectGetter.
func (c *Client) GetS3Object(ctx context.Context, bucket, key string) ([]byte, error) {
	return c.getObject(ctx, bucket, key)
}

func (c *Client) objectURL(bucket, key string) (*url.URL, error) {
	key = strings.TrimLeft(key, "/")
	key = prefixedKey(c.cfg.KeyPrefix, key)

	// Custom endpoint (MinIO, etc.): path-style https://host/bucket/key
	if c.cfg.Endpoint != "" {
		base, err := url.Parse(c.cfg.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("blobstore: parse endpoint: %w", err)
		}
		if base.Scheme == "" {
			return nil, fmt.Errorf("blobstore: endpoint must include scheme")
		}
		base.Path = strings.TrimSuffix(base.Path, "/")
		path := joinURLPath(base.Path, "/"+bucket+"/"+key)
		base.Path = path
		return base, nil
	}

	// AWS virtual-hosted-style
	u := &url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s.s3.%s.amazonaws.com", bucket, c.cfg.Region),
		Path:   "/" + key,
	}
	return u, nil
}

func joinURLPath(prefix, rest string) string {
	p := strings.TrimSuffix(prefix, "/") + rest
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// collapse duplicate slashes (except after scheme)
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

// PutObject uploads data to bucket under key (prefix applied to key).
func (c *Client) PutObject(ctx context.Context, bucket, key, contentType string, data []byte) error {
	u, err := c.objectURL(bucket, key)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	bodyHash := sha256Hex(data)
	now := time.Now().UTC()
	if err := signRequestV4(req, c.cfg, bodyHash, "s3", now); err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("blobstore: put %s/%s: status %d: %s", bucket, key, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) getObject(ctx context.Context, bucket, key string) ([]byte, error) {
	u, err := c.objectURL(bucket, key)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	emptyHash := sha256Hex(nil)
	now := time.Now().UTC()
	if err := signRequestV4(req, c.cfg, emptyHash, "s3", now); err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("blobstore: get %s/%s: status %d: %s", bucket, key, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(resp.Body)
}

func prefixedKey(prefix, key string) string {
	key = strings.TrimLeft(key, "/")
	if prefix == "" {
		return key
	}
	return strings.Trim(prefix, "/") + "/" + key
}
