package matrix

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"maunium.net/go/mautrix/id"
)

type FileSyncStore struct {
	mu      sync.Mutex
	path    string
	filters map[id.UserID]string
}

func NewFileSyncStore(path string) *FileSyncStore {
	return &FileSyncStore{
		path:    path,
		filters: make(map[id.UserID]string),
	}
}

func (s *FileSyncStore) SaveFilterID(_ context.Context, userID id.UserID, filterID string) error {
	s.mu.Lock()
	s.filters[userID] = filterID
	s.mu.Unlock()
	return nil
}

func (s *FileSyncStore) LoadFilterID(_ context.Context, userID id.UserID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.filters[userID], nil
}

func (s *FileSyncStore) SaveNextBatch(_ context.Context, _ id.UserID, nextBatchToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeFileAtomically(s.path, []byte(strings.TrimSpace(nextBatchToken)+"\n"), 0o600)
}

func (s *FileSyncStore) LoadNextBatch(_ context.Context, _ id.UserID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
