package chezmoi

// FIXME fix integration test and code
// FIXME add integration build flag

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ EncryptionTool = &GPGEncryptionTool{}

func TestGPGEncryptionTool(t *testing.T) {
	t.Skip("broken test")

	tempDir, err := ioutil.TempDir("", "chezmoi-test-gpg-encryption-tool")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tempDir))
	}()

	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(tempDir, 0o700))
	}

	et := &GPGEncryptionTool{
		Command: "gpg",
		Args: []string{
			"--batch",
			"--homedir", tempDir,
			"--passphrase", "passphrase",
			"--pinentry-mode", "loopback",
			"--no-tty",
			"--yes",
		},
		Symmetric: true,
	}
	require.NoError(t, et.runWithArgs([]string{
		"--quick-generate-key", "chezmoi",
	}))

	testEncryptionToolDecryptToFile(t, et)
	testEncryptionToolEncryptDecrypt(t, et)
	testEncryptionToolEncryptFile(t, et)
}
