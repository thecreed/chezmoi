package chezmoi

import (
	"os"
)

// WriteFile implements System.WriteFile.
func (s *RealSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return WriteFile(s.FS, filename, data, perm)
}
