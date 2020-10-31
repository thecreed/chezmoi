package chezmoi

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"

	"go.uber.org/multierr"
)

// A GPGEncryptionTool uses gpg for encryption and decryption.
type GPGEncryptionTool struct {
	Command   string
	Args      []string
	Recipient string
	Symmetric bool
}

// Decrypt implements EncyrptionTool.Decrypt.
func (t *GPGEncryptionTool) Decrypt(filenameHint string, ciphertext []byte) (plaintext []byte, err error) {
	filename, cleanup, err := t.DecryptToFile(filenameHint, ciphertext)
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, cleanup())
	}()
	return ioutil.ReadFile(filename)
}

// DecryptToFile implements EncryptionTool.DecryptToFile.
func (t *GPGEncryptionTool) DecryptToFile(filenameHint string, ciphertext []byte) (filename string, cleanupFunc CleanupFunc, err error) {
	tempDir, err := ioutil.TempDir("", "chezmoi-gpg-decrypt")
	if err != nil {
		return
	}
	cleanupFunc = func() error {
		return os.RemoveAll(tempDir)
	}

	filename = path.Join(tempDir, path.Base(filenameHint))
	inputFilename := filename + ".gpg"
	if err = ioutil.WriteFile(inputFilename, ciphertext, 0o600); err != nil {
		err = multierr.Append(err, cleanupFunc())
		return
	}

	args := []string{
		"--armor",
		"--decrypt",
		"--output", filename,
		"--quiet",
	}
	if t.Symmetric {
		args = append(args, "--symmetric")
	}
	args = append(args, inputFilename)

	if err = t.runWithArgs(args); err != nil {
		err = multierr.Append(err, cleanupFunc())
		return
	}

	return
}

// Encrypt implements EncryptionTool.Encrypt.
func (t *GPGEncryptionTool) Encrypt(plaintext []byte) (ciphertext []byte, err error) {
	tempFile, err := ioutil.TempFile("", "chezmoi-gpg-encrypt")
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, os.RemoveAll(tempFile.Name()))
	}()

	if runtime.GOOS != "windows" {
		if err = tempFile.Chmod(0o600); err != nil {
			return
		}
	}

	if err = ioutil.WriteFile(tempFile.Name(), ciphertext, 0o600); err != nil {
		return
	}

	return t.EncryptFile(tempFile.Name())
}

// EncryptFile implements EncryptionTool.EncryptFile.
func (t *GPGEncryptionTool) EncryptFile(filename string) (ciphertext []byte, err error) {
	tempDir, err := ioutil.TempDir("", "chezmoi-gpg-encrypt")
	if err != nil {
		return
	}
	defer func() {
		err = multierr.Append(err, os.RemoveAll(tempDir))
	}()

	outputFilename := path.Join(tempDir, path.Base(filename)+".gpg")
	args := []string{
		"--armor",
		"--encrypt",
		"--output", outputFilename,
		"--quiet",
	}
	switch {
	case t.Symmetric:
		args = append(args, "--symmetric")
	case t.Recipient != "":
		args = append(args, "--recipient", t.Recipient)
	}
	args = append(args, filename)

	if err = t.runWithArgs(args); err != nil {
		return
	}

	ciphertext, err = ioutil.ReadFile(outputFilename)
	return
}

func (t *GPGEncryptionTool) runWithArgs(args []string) error {
	//nolint:gosec
	cmd := exec.Command(t.Command, append(t.Args, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
