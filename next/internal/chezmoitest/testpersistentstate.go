package chezmoitest

// A PersistentState is a persistent state for testing.
type PersistentState map[string]map[string][]byte

// NewPersistentState returns a new PersistentState.
func NewPersistentState() PersistentState {
	return make(PersistentState)
}

// Delete implements PersistentState.Delete.
func (s PersistentState) Delete(bucket, key []byte) error {
	bucketMap, ok := s[string(bucket)]
	if !ok {
		return nil
	}
	delete(bucketMap, string(key))
	return nil
}

// ForEach implements PersistentState.ForEach.
func (s PersistentState) ForEach(bucket []byte, fn func(k, v []byte) error) error {
	for k, v := range s[string(bucket)] {
		if err := fn([]byte(k), v); err != nil {
			return err
		}
	}
	return nil
}

// Get implements PersistentState.Get.
func (s PersistentState) Get(bucket, key []byte) ([]byte, error) {
	bucketMap, ok := s[string(bucket)]
	if !ok {
		return nil, nil
	}
	return bucketMap[string(key)], nil
}

// OpenOrCreate implements PersistentState.OpenOrCreate.
func (s PersistentState) OpenOrCreate() error {
	return nil
}

// Set implements PersistentState.Set.
func (s PersistentState) Set(bucket, key, value []byte) error {
	bucketMap, ok := s[string(bucket)]
	if !ok {
		bucketMap = make(map[string][]byte)
		s[string(bucket)] = bucketMap
	}
	bucketMap[string(key)] = value
	return nil
}
