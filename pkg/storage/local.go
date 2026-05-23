package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Local struct {
	root      string
	publicURL string
}

func NewLocal(root, publicURL string) *Local {
	return &Local{root: root, publicURL: strings.TrimRight(publicURL, "/")}
}

func (l *Local) Put(ctx context.Context, object Object) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	key := object.Key
	if key == "" {
		key = randomKey()
	}
	path, err := l.safePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, object.Reader)
	if err != nil {
		return err
	}
	return nil
}

func (l *Local) Stat(ctx context.Context, key string) (StoredObject, error) {
	if err := ctx.Err(); err != nil {
		return StoredObject{}, err
	}
	path, err := l.safePath(key)
	if err != nil {
		return StoredObject{}, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return StoredObject{}, err
	}

	stored := StoredObject{
		Key:       key,
		Size:      stat.Size(),
		CreatedAt: stat.ModTime().UTC(),
	}
	if l.publicURL != "" {
		stored.URL = l.publicURL + "/" + url.PathEscape(key)
	}
	return stored, nil
}

func (l *Local) Get(ctx context.Context, key string) (Object, error) {
	if err := ctx.Err(); err != nil {
		return Object{}, err
	}
	path, err := l.safePath(key)
	if err != nil {
		return Object{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return Object{}, err
	}
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return Object{}, err
	}
	return Object{Key: key, Reader: file, Size: stat.Size()}, nil
}

func (l *Local) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := l.safePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (l *Local) SignedURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	if l.publicURL == "" {
		return "", ErrUnsupported
	}
	expires := time.Now().UTC().Add(ttl).Unix()
	return l.publicURL + "/" + url.PathEscape(key) + "?expires=" + url.QueryEscape(time.Unix(expires, 0).Format(time.RFC3339)), nil
}

func (l *Local) safePath(key string) (string, error) {
	clean := filepath.Clean(strings.TrimPrefix(key, "/"))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", errors.New("invalid object key")
	}
	return filepath.Join(l.root, clean), nil
}

func randomKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b[:])
}
