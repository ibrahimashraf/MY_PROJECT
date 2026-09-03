package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type Object struct {
	Key         string
	ContentType string
	Data        []byte
	Metadata    map[string]string
}

type Store interface {
	Put(ctx context.Context, object Object) error
	Get(ctx context.Context, key string) (Object, error)
	Delete(ctx context.Context, key string) error
	Health(ctx context.Context) error
}

type InMemoryStore struct {
	mu     sync.RWMutex
	bucket map[string]Object
}

func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{bucket: make(map[string]Object)} }
func (s *InMemoryStore) Put(ctx context.Context, object Object) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateObject(object); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	object.Data = append([]byte(nil), object.Data...)
	object.Metadata = cloneMetadata(object.Metadata)
	s.bucket[object.Key] = object
	return nil
}
func (s *InMemoryStore) Get(ctx context.Context, key string) (Object, error) {
	if err := ctx.Err(); err != nil {
		return Object{}, err
	}
	if strings.TrimSpace(key) == "" {
		return Object{}, errors.New("object key is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	object, ok := s.bucket[key]
	if !ok {
		return Object{}, errors.New("object not found")
	}
	object.Data = append([]byte(nil), object.Data...)
	object.Metadata = cloneMetadata(object.Metadata)
	return object, nil
}
func (s *InMemoryStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(key) == "" {
		return errors.New("object key is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.bucket[key]; !ok {
		return errors.New("object not found")
	}
	delete(s.bucket, key)
	return nil
}
func (s *InMemoryStore) Health(ctx context.Context) error { return ctx.Err() }

// HTTPDoer and Signer keep the S3 adapter testable and allow deployment to
// supply AWS Signature V4 signing without coupling the domain to an SDK.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type Signer func(*http.Request) error

// NewAWSSigV4Signer returns a small, self-contained Signature V4 signer for
// RustFS and other S3-compatible endpoints. It signs only stable transport
// headers and the exact payload digest already bound to the request.
func NewAWSSigV4Signer(accessKey, secretKey, region, service string) Signer {
	return func(request *http.Request) error {
		if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" || strings.TrimSpace(region) == "" || strings.TrimSpace(service) == "" {
			return errors.New("storage signing credentials, region, and service are required")
		}
		now := time.Now().UTC()
		amzDate := now.Format("20060102T150405Z")
		shortDate := now.Format("20060102")
		payloadHash := request.Header.Get("X-Amz-Content-Sha256")
		if payloadHash == "" {
			payloadHash = sha256Hex(nil)
			request.Header.Set("X-Amz-Content-Sha256", payloadHash)
		}
		request.Header.Set("X-Amz-Date", amzDate)
		host := request.Host
		if host == "" {
			host = request.URL.Host
		}
		var metaKeys []string
		for k := range request.Header {
			lowerK := strings.ToLower(k)
			if strings.HasPrefix(lowerK, "x-amz-meta-") {
				metaKeys = append(metaKeys, lowerK)
			}
		}
		sort.Strings(metaKeys)

		headerList := []string{"host", "x-amz-content-sha256", "x-amz-date"}
		headerList = append(headerList, metaKeys...)
		
		var canon []string
		for _, k := range headerList {
			if k == "host" {
				canon = append(canon, "host:"+strings.TrimSpace(host))
			} else if k == "x-amz-content-sha256" {
				canon = append(canon, "x-amz-content-sha256:"+payloadHash)
			} else if k == "x-amz-date" {
				canon = append(canon, "x-amz-date:"+amzDate)
			} else {
				val := strings.TrimSpace(request.Header.Get(k))
				canon = append(canon, k+":"+val)
			}
		}

		signedHeaders := strings.Join(headerList, ";")
		canonicalHeaders := strings.Join(canon, "\n") + "\n"
		canonicalRequest := strings.Join([]string{request.Method, canonicalURI(request.URL), canonicalQuery(request.URL), canonicalHeaders, signedHeaders, payloadHash}, "\n")
		scope := shortDate + "/" + region + "/" + service + "/aws4_request"
		stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", amzDate, scope, sha256Hex([]byte(canonicalRequest))}, "\n")
		signingKey := hmacSHA256(hmacSHA256(hmacSHA256(hmacSHA256([]byte("AWS4"+secretKey), shortDate), region), service), "aws4_request")
		signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
		request.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+accessKey+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
		return nil
	}
}

func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func canonicalURI(value *url.URL) string {
	path := value.EscapedPath()
	if path == "" {
		return "/"
	}
	return path
}

func canonicalQuery(value *url.URL) string {
	query := value.Query()
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0)
	for _, key := range keys {
		for _, item := range query[key] {
			values = append(values, url.QueryEscape(key)+"="+url.QueryEscape(item))
		}
	}
	return strings.Join(values, "&")
}

type S3Store struct {
	Endpoint *url.URL
	Bucket   string
	Client   HTTPDoer
	Sign     Signer
}

func NewS3Store(endpoint, bucket string, client HTTPDoer, signer Signer) (*S3Store, error) {
	if strings.TrimSpace(endpoint) == "" || strings.TrimSpace(bucket) == "" {
		return nil, errors.New("endpoint and bucket are required")
	}
	parsed, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("valid storage endpoint is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	if signer == nil {
		return nil, errors.New("storage request signer is required")
	}
	return &S3Store{Endpoint: parsed, Bucket: bucket, Client: client, Sign: signer}, nil
}

// MinIOStore and NewMinIOStore remain compatibility aliases for existing
// callers. The implementation is a vendor-neutral S3 client and is used for
// RustFS, SeaweedFS, MinIO-compatible, and other S3 endpoints.
type MinIOStore = S3Store

func NewMinIOStore(endpoint, bucket string, client HTTPDoer, signer Signer) (*S3Store, error) {
	return NewS3Store(endpoint, bucket, client, signer)
}

func (s *S3Store) objectURL(key string) (*url.URL, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("object key is required")
	}
	if strings.HasPrefix(key, "/") || strings.Contains(key, "..") {
		return nil, errors.New("invalid object key")
	}
	copied := *s.Endpoint
	copied.Path = strings.TrimRight(copied.Path, "/") + "/" + url.PathEscape(s.Bucket) + "/" + escapePath(key)
	return &copied, nil
}
func escapePath(value string) string {
	parts := strings.Split(value, "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
func (s *S3Store) Put(ctx context.Context, object Object) error {
	if err := validateObject(object); err != nil {
		return err
	}
	endpoint, err := s.objectURL(object.Key)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), bytes.NewReader(object.Data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", object.ContentType)
	request.Header.Set("X-Amz-Content-Sha256", sha256Hex(object.Data))
	for key, value := range object.Metadata {
		request.Header.Set("X-Amz-Meta-"+key, value)
	}
	if err := s.Sign(request); err != nil {
		return err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("object put failed with status %d", response.StatusCode)
	}
	return nil
}
func (s *S3Store) Get(ctx context.Context, key string) (Object, error) {
	endpoint, err := s.objectURL(key)
	if err != nil {
		return Object{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Object{}, err
	}
	if err := s.Sign(request); err != nil {
		return Object{}, err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return Object{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Object{}, fmt.Errorf("object get failed with status %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return Object{}, err
	}
	metadata := make(map[string]string)
	for header, values := range response.Header {
		prefix := "x-amz-meta-"
		lowered := strings.ToLower(header)
		if !strings.HasPrefix(lowered, prefix) {
			continue
		}
		metadata[strings.TrimPrefix(lowered, prefix)] = strings.Join(values, ",")
	}
	return Object{Key: key, ContentType: response.Header.Get("Content-Type"), Data: data, Metadata: metadata}, nil
}
func (s *S3Store) Delete(ctx context.Context, key string) error {
	endpoint, err := s.objectURL(key)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint.String(), nil)
	if err != nil {
		return err
	}
	if err := s.Sign(request); err != nil {
		return err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("object delete failed with status %d", response.StatusCode)
	}
	return nil
}
func (s *S3Store) Health(ctx context.Context) error {
	endpoint, err := s.objectURL("health-check")
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint.String(), nil)
	if err != nil {
		return err
	}
	if err := s.Sign(request); err != nil {
		return err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 500 {
		return fmt.Errorf("storage health failed with status %d", response.StatusCode)
	}
	return nil
}
func validateObject(object Object) error {
	if strings.TrimSpace(object.Key) == "" {
		return errors.New("object key is required")
	}
	if strings.HasPrefix(object.Key, "/") || strings.Contains(object.Key, "..") {
		return errors.New("invalid object key")
	}
	if strings.TrimSpace(object.ContentType) == "" {
		return errors.New("content type is required")
	}
	return nil
}
func cloneMetadata(metadata map[string]string) map[string]string {
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
