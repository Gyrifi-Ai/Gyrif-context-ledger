package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type ObjectStore struct{ root string }

func NewObjectStore(root string) (*ObjectStore, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create object directory: %w", err)
	}
	return &ObjectStore{root: root}, nil
}

func (store *ObjectStore) Write(ctx context.Context, kind string, value []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	digest := sha256.Sum256(append(append([]byte(kind), 0), value...))
	hash := hex.EncodeToString(digest[:])
	directory := filepath.Join(store.root, hash[:2])
	path := filepath.Join(directory, hash[2:])
	if _, err := os.Stat(path); err == nil {
		return "sha256:" + hash, nil
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", fmt.Errorf("create object shard: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".gyrifi-object-*")
	if err != nil {
		return "", fmt.Errorf("create temporary object: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o640); err != nil {
		temporary.Close()
		return "", fmt.Errorf("set object mode: %w", err)
	}
	if _, err := temporary.Write(value); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write object: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return "", fmt.Errorf("sync object: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close object: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("publish object: %w", err)
	}
	return "sha256:" + hash, nil
}

func (store *ObjectStore) Read(ctx context.Context, hash string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	hash = strings.TrimPrefix(hash, "sha256:")
	if len(hash) != 64 {
		return nil, fmt.Errorf("invalid object hash")
	}
	value, err := os.ReadFile(filepath.Join(store.root, hash[:2], hash[2:]))
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}
	return value, nil
}

func (store *ObjectStore) Size(ctx context.Context) (int64, error) {
	var total int64
	err := filepath.WalkDir(store.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".gyrifi-object-") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("measure object store: %w", err)
	}
	return total, nil
}
