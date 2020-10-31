package chezmoi

import (
	"log"
	"os"
)

// A PersistentState is a persistent state.
type PersistentState interface {
	Get(bucket, key []byte) ([]byte, error)
	Delete(bucket, key []byte) error
	ForEach(bucket []byte, fn func(k, v []byte) error) error
	OpenOrCreate() error
	Set(bucket, key, value []byte) error
}

type debugPersistentState struct {
	s      PersistentState
	logger *log.Logger
}

type dryRunPersistentState struct {
	s        PersistentState
	modified bool
}

type nullPersistentState struct{}

type readOnlyPersistentState struct {
	s PersistentState
}

func newDebugPersistentState(s PersistentState, logger *log.Logger) *debugPersistentState {
	return &debugPersistentState{
		s:      s,
		logger: logger,
	}
}

func (s *debugPersistentState) Delete(bucket, key []byte) error {
	return s.debugf("Delete(%q, %q)", []interface{}{string(bucket), string(key)}, func() error {
		return s.s.Delete(bucket, key)
	})
}

func (s *debugPersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	return s.debugf("ForEach(%q, _)", []interface{}{string(bucket)}, func() error {
		return s.s.ForEach(bucket, fn)
	})
}

func (s *debugPersistentState) Get(bucket, key []byte) ([]byte, error) {
	var value []byte
	err := s.debugf("Get(%q, %q)", []interface{}{string(bucket), string(key)}, func() error {
		var err error
		value, err = s.s.Get(bucket, key)
		return err
	})
	return value, err
}

func (s *debugPersistentState) OpenOrCreate() error {
	return s.debugf("OpenOrCreate", nil, s.s.OpenOrCreate)
}

func (s *debugPersistentState) Set(bucket, key, value []byte) error {
	return s.debugf("Set(%q, %q, %q)", []interface{}{string(bucket), string(key), string(value)}, func() error {
		return s.s.Set(bucket, key, value)
	})
}

func (s *debugPersistentState) debugf(format string, args []interface{}, f func() error) error {
	err := f()
	if err != nil {
		s.logger.Printf(format+" == %v", append(args, err))
	} else {
		s.logger.Printf(format, args...)
	}
	return err
}

func newDryRunPersistentState(s PersistentState) *dryRunPersistentState {
	return &dryRunPersistentState{
		s: s,
	}
}

func (s *dryRunPersistentState) Get(bucket, key []byte) ([]byte, error) {
	return s.s.Get(bucket, key)
}

func (s *dryRunPersistentState) Delete(bucket, key []byte) error {
	s.modified = true
	return nil
}

func (s *dryRunPersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	return s.s.ForEach(bucket, fn)
}

func (s *dryRunPersistentState) OpenOrCreate() error {
	s.modified = true // FIXME this will give false naegatives if s.s already exists, need to separate create from open
	return s.s.OpenOrCreate()
}

func (s *dryRunPersistentState) Set(bucket, key, value []byte) error {
	s.modified = true
	// FIXME do we need to remember that the value has been set?
	return nil
}

func (nullPersistentState) Get(bucket, key []byte) ([]byte, error)                  { return nil, nil }
func (nullPersistentState) Delete(bucket, key []byte) error                         { return nil }
func (nullPersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error { return nil }
func (nullPersistentState) OpenOrCreate() error                                     { return nil }
func (nullPersistentState) Set(bucket, key, value []byte) error                     { return nil }

func newReadOnlyPersistentState(s PersistentState) PersistentState {
	return &readOnlyPersistentState{
		s: s,
	}
}

func (s *readOnlyPersistentState) Get(bucket, key []byte) ([]byte, error) {
	return s.s.Get(bucket, key)
}

func (s *readOnlyPersistentState) Delete(bucket, key []byte) error {
	return os.ErrPermission
}

func (s *readOnlyPersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	return s.s.ForEach(bucket, fn)
}

func (s *readOnlyPersistentState) OpenOrCreate() error {
	return s.s.OpenOrCreate()
}

func (s *readOnlyPersistentState) Set(bucket, key, value []byte) error {
	return os.ErrPermission
}
