package custom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileDB struct {
	dir     string
	dbPath  string
	mu      sync.RWMutex
	initErr error
}

func NewCustomKeychain(email string) *FileDB {
	baseDir, err := os.Getwd() // 当前工作目录（常见理解的“当前应用程序目录”）
	if err != nil {
		baseDir = "."
	}

	dirName := strings.ToLower(email)
	first := dirName[:1]

	dir := filepath.Join(baseDir, "UserData", first, dirName)

	//safeUser := sanitizePathPart(username)
	//dir := filepath.Join(baseDir, "UserData", safeUser)

	db := &FileDB{
		dir:    dir,
		dbPath: filepath.Join(dir, "db.json"),
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		db.initErr = err
		return db
	}

	// 保证 db.json 至少是个合法 JSON（空对象）
	if _, err := os.Stat(db.dbPath); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(db.dbPath, []byte("{}\n"), 0o644)
	}

	return db
}

func (f *FileDB) Get(key string) ([]byte, error) {
	if f == nil {
		return nil, errors.New("FileDB is nil")
	}
	if f.initErr != nil {
		return nil, f.initErr
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("key is empty")
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	db, err := f.loadLocked()
	if err != nil {
		return nil, err
	}

	val, ok := db[key]
	if !ok {
		return nil, fmt.Errorf("%w: key %q", os.ErrNotExist, key)
	}

	out := make([]byte, len(val))
	copy(out, val)
	return out, nil
}

func (f *FileDB) Set(key string, data []byte) error {
	if f == nil {
		return errors.New("FileDB is nil")
	}
	if f.initErr != nil {
		return f.initErr
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is empty")
	}
	if !json.Valid(data) {
		return errors.New("data is not valid JSON")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	db, err := f.loadLocked()
	if err != nil {
		return err
	}

	db[key] = json.RawMessage(append([]byte(nil), data...))
	return f.saveLocked(db)
}

func (f *FileDB) Remove(key string) error {
	if f == nil {
		return errors.New("FileDB is nil")
	}
	if f.initErr != nil {
		return f.initErr
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key is empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	db, err := f.loadLocked()
	if err != nil {
		return err
	}
	if _, ok := db[key]; !ok {
		return fmt.Errorf("%w: key %q", os.ErrNotExist, key)
	}

	delete(db, key)
	return f.saveLocked(db)
}

func (f *FileDB) loadLocked() (map[string]json.RawMessage, error) {
	b, err := os.ReadFile(f.dbPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]json.RawMessage{}, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return map[string]json.RawMessage{}, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("invalid db.json (%s): %w", f.dbPath, err)
	}
	if m == nil {
		m = map[string]json.RawMessage{}
	}
	return m, nil
}

func (f *FileDB) saveLocked(m map[string]json.RawMessage) error {
	if err := os.MkdirAll(f.dir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(f.dir, "db-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	// 原子替换（Windows 下可能需要先删除目标文件）
	if err := os.Rename(tmpName, f.dbPath); err == nil {
		return nil
	}
	_ = os.Remove(f.dbPath)
	return os.Rename(tmpName, f.dbPath)
}

func sanitizePathPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "default"
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '@', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
