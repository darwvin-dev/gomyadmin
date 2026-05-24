package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// S3Store implements Storage and Inspector for any S3-compatible object store
// (AWS S3, Cloudflare R2, MinIO, etc.) using only the standard library.
// Authentication uses AWS Signature Version 4.
type S3Store struct {
	config S3Config
	client *http.Client
}

// NewS3 creates an S3Store from the provided config.
// Region defaults to "us-east-1" when empty.
// ForcePathStyle or a non-empty Endpoint both enable path-style URLs.
func NewS3(config S3Config) (*S3Store, error) {
	if config.Bucket == "" {
		return nil, errors.New("S3Config.Bucket is required")
	}
	if config.AccessKeyID == "" || config.SecretAccessKey == "" {
		return nil, errors.New("S3Config.AccessKeyID and SecretAccessKey are required")
	}
	if config.Region == "" {
		config.Region = "us-east-1"
	}
	return &S3Store{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (s *S3Store) baseEndpoint() string {
	if s.config.Endpoint != "" {
		return strings.TrimRight(s.config.Endpoint, "/")
	}
	return "https://s3." + s.config.Region + ".amazonaws.com"
}

func (s *S3Store) objectURL(key string) string {
	key = strings.TrimLeft(key, "/")
	if s.config.ForcePathStyle || s.config.Endpoint != "" {
		return s.baseEndpoint() + "/" + s.config.Bucket + "/" + key
	}
	parsed, err := url.Parse(s.baseEndpoint())
	if err != nil || parsed.Host == "" {
		return s.baseEndpoint() + "/" + s.config.Bucket + "/" + key
	}
	parsed.Host = s.config.Bucket + "." + parsed.Host
	return parsed.String() + "/" + key
}

func (s *S3Store) Put(ctx context.Context, object Object) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var body []byte
	var err error
	if object.Reader != nil {
		body, err = io.ReadAll(object.Reader)
		if err != nil {
			return fmt.Errorf("reading upload body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.objectURL(object.Key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	if object.ContentType != "" {
		req.Header.Set("Content-Type", object.ContentType)
	}
	s.signRequest(req, body, time.Now().UTC())

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return s3ResponseError(resp)
	}
	return nil
}

func (s *S3Store) Get(ctx context.Context, key string) (Object, error) {
	if err := ctx.Err(); err != nil {
		return Object{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key), nil)
	if err != nil {
		return Object{}, err
	}
	s.signRequest(req, nil, time.Now().UTC())

	resp, err := s.client.Do(req)
	if err != nil {
		return Object{}, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return Object{}, ErrNotFound
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return Object{}, s3ResponseError(resp)
	}
	return Object{
		Key:         key,
		Reader:      resp.Body,
		ContentType: resp.Header.Get("Content-Type"),
		Size:        resp.ContentLength,
	}, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(key), nil)
	if err != nil {
		return err
	}
	s.signRequest(req, nil, time.Now().UTC())

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return s3ResponseError(resp)
}

func (s *S3Store) Stat(ctx context.Context, key string) (StoredObject, error) {
	if err := ctx.Err(); err != nil {
		return StoredObject{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.objectURL(key), nil)
	if err != nil {
		return StoredObject{}, err
	}
	s.signRequest(req, nil, time.Now().UTC())

	resp, err := s.client.Do(req)
	if err != nil {
		return StoredObject{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return StoredObject{}, ErrNotFound
	}
	if resp.StatusCode >= 300 {
		return StoredObject{}, fmt.Errorf("S3 HEAD failed with status %d", resp.StatusCode)
	}
	stored := StoredObject{
		Key:         key,
		ContentType: resp.Header.Get("Content-Type"),
		Size:        resp.ContentLength,
	}
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, parseErr := http.ParseTime(lm); parseErr == nil {
			stored.CreatedAt = t.UTC()
		}
	}
	if s.config.PublicBaseURL != "" {
		stored.URL = strings.TrimRight(s.config.PublicBaseURL, "/") + "/" + strings.TrimLeft(key, "/")
	}
	return stored, nil
}

// SignedURL returns a presigned GET URL valid for the given TTL.
// The URL allows unauthenticated download of a private object.
func (s *S3Store) SignedURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	date := now.Format("20060102")
	datetime := now.Format("20060102T150405Z")
	credential := s.config.AccessKeyID + "/" + date + "/" + s.config.Region + "/s3/aws4_request"
	expires := fmt.Sprintf("%d", int(ttl.Seconds()))

	rawURL := s.objectURL(key)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	q := parsed.Query()
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Credential", credential)
	q.Set("X-Amz-Date", datetime)
	q.Set("X-Amz-Expires", expires)
	q.Set("X-Amz-SignedHeaders", "host")
	parsed.RawQuery = sortedQueryString(q)

	canonicalRequest := strings.Join([]string{
		http.MethodGet,
		parsed.EscapedPath(),
		parsed.RawQuery,
		"host:" + parsed.Host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")

	scope := date + "/" + s.config.Region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + datetime + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	sig := hmacHex(s.signingKey(date), stringToSign)

	parsed.RawQuery += "&X-Amz-Signature=" + url.QueryEscape(sig)
	return parsed.String(), nil
}

// signRequest adds AWS Signature V4 authorization to req.
// body may be nil for requests with no body (GET, HEAD, DELETE).
func (s *S3Store) signRequest(req *http.Request, body []byte, now time.Time) {
	date := now.Format("20060102")
	datetime := now.Format("20060102T150405Z")

	bodyHash := emptySHA256
	if len(body) > 0 {
		bodyHash = sha256Hex(body)
	}

	req.Header.Set("X-Amz-Date", datetime)
	req.Header.Set("X-Amz-Content-Sha256", bodyHash)

	// Build the canonical headers map: lowercase name → trimmed value.
	// The host header must be included; Go's http.Client sets it from req.URL.Host.
	hdrs := make(map[string]string, len(req.Header)+1)
	for name, values := range req.Header {
		hdrs[strings.ToLower(name)] = strings.TrimSpace(strings.Join(values, ","))
	}
	host := req.URL.Host
	if req.Host != "" {
		host = req.Host
	}
	hdrs["host"] = host

	names := make([]string, 0, len(hdrs))
	for name := range hdrs {
		names = append(names, name)
	}
	sort.Strings(names)

	var canonicalHeaders strings.Builder
	for _, name := range names {
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteByte(':')
		canonicalHeaders.WriteString(hdrs[name])
		canonicalHeaders.WriteByte('\n')
	}
	signedHeaders := strings.Join(names, ";")

	canonicalPath := req.URL.EscapedPath()
	if canonicalPath == "" {
		canonicalPath = "/"
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalPath,
		sortedQueryString(req.URL.Query()),
		canonicalHeaders.String(),
		signedHeaders,
		bodyHash,
	}, "\n")

	scope := date + "/" + s.config.Region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + datetime + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	sig := hmacHex(s.signingKey(date), stringToSign)

	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential="+s.config.AccessKeyID+"/"+scope+
			", SignedHeaders="+signedHeaders+
			", Signature="+sig)
}

func (s *S3Store) signingKey(date string) []byte {
	kDate := hmacBytes([]byte("AWS4"+s.config.SecretAccessKey), date)
	kRegion := hmacBytes(kDate, s.config.Region)
	kService := hmacBytes(kRegion, "s3")
	return hmacBytes(kService, "aws4_request")
}

// sortedQueryString returns a percent-encoded query string with keys sorted lexicographically.
// AWS SigV4 requires this exact ordering for canonical query strings.
func sortedQueryString(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(q))
	for _, k := range keys {
		vals := append([]string(nil), q[k]...)
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacBytes(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hmacHex(key []byte, data string) string {
	return hex.EncodeToString(hmacBytes(key, data))
}

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

func s3ResponseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var s3Err s3ErrorResponse
	if xmlErr := xml.Unmarshal(body, &s3Err); xmlErr == nil && s3Err.Code != "" {
		return fmt.Errorf("S3 %s: %s", s3Err.Code, s3Err.Message)
	}
	return fmt.Errorf("S3 request failed with status %d", resp.StatusCode)
}
