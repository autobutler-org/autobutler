package fileutil

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/nwaples/rardecode/v2"
)

// Go cannot write an encrypted rar or 7z, so the errors their readers give
// for a missing password are checked here, wrapped the way archiver hands
// them back, rather than through a fixture.
func TestEncryptedOrRecognizesEveryFormatsPasswordError(t *testing.T) {
	for _, err := range []error{
		rardecode.ErrArchiveEncrypted,
		rardecode.ErrArchivedFileEncrypted,
		fmt.Errorf("handling file: a.txt: %w", rardecode.ErrBadPassword),
		errors.New("aes7z: no password set"),
	} {
		got := encryptedOr("secret.rar", err)
		var encrypted *EncryptedArchiveError
		var unsupported *UnsupportedError
		if !errors.As(got, &unsupported) || !errors.As(got, &encrypted) || encrypted.Name != "secret.rar" {
			t.Errorf("encryptedOr(%v) = %T %v, want an EncryptedArchiveError naming secret.rar", err, got, got)
		}
	}
	for _, err := range []error{nil, io.ErrUnexpectedEOF, rardecode.ErrCorruptFileHeader} {
		if got := encryptedOr("x.rar", err); got != err {
			t.Errorf("encryptedOr(%v) = %v, want it unchanged", err, got)
		}
	}
}
