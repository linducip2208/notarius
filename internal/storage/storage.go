// Package storage abstracts private file storage with an S3-compatible adapter.
package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Store saves and opens private files by reference. Nothing stored here
// is ever served as a public file; handlers authorize every download.
type Store interface {
	Save(ref string, r io.Reader) error
	Open(ref string) (io.ReadCloser, error)
	Delete(ref string) error
}

// LocalStore keeps files under a base directory with safe path joining.
type LocalStore struct{ base string }

func NewLocal(base string) (*LocalStore, error) {
	if err := os.MkdirAll(base, 0o750); err != nil {
		return nil, err
	}
	return &LocalStore{base: base}, nil
}

func (l *LocalStore) path(ref string) (string, error) {
	clean := filepath.Clean("/" + ref)
	if clean == "/" || len(clean) > 512 {
		return "", errors.New("invalid file reference")
	}
	return filepath.Join(l.base, clean), nil
}

func (l *LocalStore) Save(ref string, r io.Reader) error {
	p, err := l.path(ref)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, (100<<20)+1)) // 100MB cap, +1 probe
	if err != nil {
		f.Close()
		os.Remove(p)
		return err
	}
	if n > 100<<20 {
		f.Close()
		os.Remove(p)
		return errors.New("berkas melebihi batas 100MB")
	}
	return f.Close()
}

func (l *LocalStore) Open(ref string) (io.ReadCloser, error) {
	p, err := l.path(ref)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Delete removes one object (idempotent).
func (l *LocalStore) Delete(ref string) error {
	p, err := l.path(ref)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
