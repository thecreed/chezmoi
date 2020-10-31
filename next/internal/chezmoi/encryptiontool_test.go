package chezmoi

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

var _ EncryptionTool = &nullEncryptionTool{}

type testEncryptionTool struct {
	key byte
}

var _ EncryptionTool = &testEncryptionTool{}

type testEncryptionToolOption func(*testEncryptionTool)

func newTestEncryptionTool(options ...testEncryptionToolOption) *testEncryptionTool {
	t := &testEncryptionTool{
		//nolint:gosec
		key: byte(rand.Int() + 1),
	}
	for _, option := range options {
		option(t)
	}
	return t
}

func (t *testEncryptionTool) Decrypt(filenameHint string, ciphertext []byte) ([]byte, error) {
	return t.xorWithKey(ciphertext), nil
}

func (t *testEncryptionTool) DecryptToFile(filenameHint string, ciphertext []byte) (filename string, cleanupFunc CleanupFunc, err error) {
	tempDir, err := ioutil.TempDir("", "chezmoi-test-decrypt")
	if err != nil {
		return
	}
	cleanupFunc = func() error {
		return os.RemoveAll(tempDir)
	}

	filename = path.Join(tempDir, path.Base(filenameHint))
	if err = ioutil.WriteFile(filename, t.xorWithKey(ciphertext), 0o600); err != nil {
		err = multierr.Append(err, cleanupFunc())
		return
	}

	return
}

func (t *testEncryptionTool) Encrypt(plaintext []byte) ([]byte, error) {
	return t.xorWithKey(plaintext), nil
}

func (t *testEncryptionTool) EncryptFile(filename string) ([]byte, error) {
	plaintext, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return t.xorWithKey(plaintext), nil
}

func (t *testEncryptionTool) xorWithKey(input []byte) []byte {
	output := make([]byte, 0, len(input))
	for _, b := range input {
		output = append(output, b^t.key)
	}
	return output
}

func testEncryptionToolDecryptToFile(t *testing.T, et EncryptionTool) {
	t.Run("DecryptToFile", func(t *testing.T) {
		expectedPlaintext := []byte("secret")

		actualCiphertext, err := et.Encrypt(expectedPlaintext)
		require.NoError(t, err)
		assert.NotEqual(t, expectedPlaintext, actualCiphertext)

		filenameHint := "filename.txt"
		filename, cleanup, err := et.DecryptToFile(filenameHint, actualCiphertext)
		require.NoError(t, err)
		assert.True(t, strings.Contains(filename, filenameHint))
		assert.NotNil(t, cleanup)
		defer func() {
			assert.NoError(t, cleanup())
		}()

		actualPlaintext, err := ioutil.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, expectedPlaintext, actualPlaintext)
	})
}

func testEncryptionToolEncryptDecrypt(t *testing.T, et EncryptionTool) {
	t.Run("EncryptDecrypt", func(t *testing.T) {
		expectedPlaintext := []byte("secret")

		actualCiphertext, err := et.Encrypt(expectedPlaintext)
		require.NoError(t, err)
		assert.NotEqual(t, expectedPlaintext, actualCiphertext)

		actualPlaintext, err := et.Decrypt("", actualCiphertext)
		require.NoError(t, err)
		assert.Equal(t, expectedPlaintext, actualPlaintext)
	})
}

func testEncryptionToolEncryptFile(t *testing.T, et EncryptionTool) {
	t.Run("EncryptFile", func(t *testing.T) {
		expectedPlaintext := []byte("secret")

		tempFile, err := ioutil.TempFile("", "chezmoi-test-encryption-tool")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, os.RemoveAll(tempFile.Name()))
		}()
		if runtime.GOOS != "windows" {
			require.NoError(t, tempFile.Chmod(0o600))
		}
		_, err = tempFile.Write(expectedPlaintext)
		require.NoError(t, err)
		require.NoError(t, tempFile.Close())

		actualCiphertext, err := et.EncryptFile(tempFile.Name())
		require.NoError(t, err)
		assert.NotEqual(t, expectedPlaintext, actualCiphertext)

		actualPlaintext, err := et.Decrypt("", actualCiphertext)
		require.NoError(t, err)
		assert.Equal(t, expectedPlaintext, actualPlaintext)
	})
}

func TestTestEncruptionTool(t *testing.T) {
	et := newTestEncryptionTool()
	testEncryptionToolDecryptToFile(t, et)
	testEncryptionToolEncryptDecrypt(t, et)
	testEncryptionToolEncryptFile(t, et)
}
