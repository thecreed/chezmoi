package chezmoi

import (
	"os"
	"path/filepath"

	vfs "github.com/twpayne/go-vfs"
	"go.etcd.io/bbolt"
)

// A BoltPersistentState is a state persisted with bolt.
type BoltPersistentState struct {
	fs      vfs.FS
	path    string
	options *bbolt.Options
	db      *bbolt.DB
}

// NewBoltPersistentState returns a new BoltPersistentState.
func NewBoltPersistentState(fs vfs.FS, path string, options *bbolt.Options) (*BoltPersistentState, error) {
	b := &BoltPersistentState{
		fs:      fs,
		path:    path,
		options: options,
	}
	_, err := fs.Stat(b.path)
	switch {
	case err == nil:
		if err := b.OpenOrCreate(); err != nil {
			return nil, err
		}
	case os.IsNotExist(err):
	default:
		return nil, err
	}
	return b, nil
}

// Close closes b.
func (b *BoltPersistentState) Close() error {
	if b.db == nil {
		return nil
	}
	if err := b.db.Close(); err != nil {
		return err
	}
	b.db = nil
	return nil
}

// Delete deletes the value associate with key in bucket. If bucket or key does
// not exist then Delete does nothing.
func (b *BoltPersistentState) Delete(bucket, key []byte) error {
	if b.db == nil {
		return nil
	}
	return b.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		return b.Delete(key)
	})
}

// Get returns the value associated with key in bucket.
func (b *BoltPersistentState) Get(bucket, key []byte) ([]byte, error) {
	if b.db == nil {
		return nil, nil
	}
	var value []byte
	if err := b.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		if v := b.Get(key); v != nil {
			value = make([]byte, len(v))
			copy(value, v)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return value, nil
}

// ForEach calls fn for each key, value pair in bucket.
func (b *BoltPersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	if b.db == nil {
		return nil
	}
	return b.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		return b.ForEach(fn)
	})
}

// OpenOrCreate opens b, creating it if needed.
func (b *BoltPersistentState) OpenOrCreate() error {
	if err := vfs.MkdirAll(b.fs, filepath.Dir(b.path), 0o777); err != nil {
		return err
	}
	var options bbolt.Options
	if b.options != nil {
		options = *b.options
	}
	options.OpenFile = b.fs.OpenFile
	db, err := bbolt.Open(b.path, 0o600, &options)
	if err != nil {
		return err
	}
	b.db = db
	return err
}

// Set sets the value associated with key in bucket. bucket will be created if
// it does not already exist.
func (b *BoltPersistentState) Set(bucket, key, value []byte) error {
	if b.db == nil {
		if err := b.OpenOrCreate(); err != nil {
			return err
		}
	}
	return b.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
	})
}
