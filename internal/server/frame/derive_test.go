package frame_test

import (
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"golang.org/x/crypto/argon2"
)

// referenceAuthKey is the construction the server and the app agree on, written out in Go: Argon2id13 over the
// password with 3 passes and 64 MiB, then HKDF-SHA256 with no salt and the info string "auth" (#2430).
func referenceAuthKey(t *testing.T, password string, salt []byte) string {
	t.Helper()
	master := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	key, err := hkdf.Key(sha256.New, master, nil, "auth", 32)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

// pageDerive runs the real frame.js against the app's own sodium.js under Node and returns the auth key it derives
// for each password. It skips the test when Node or sodium.js is missing, so a machine without Node still runs every
// other check.
func pageDerive(t *testing.T, saltBase64 string, passwords ...string) []string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	if _, err := os.Stat("../../../web/sodium.js"); err != nil {
		t.Skipf("web/sodium.js is not present: %v", err)
	}
	const script = `
const vm = require('vm'), fs = require('fs'), crypto = require('crypto');
const box = { console, Uint8Array, TextEncoder, TextDecoder, WebAssembly, setTimeout, clearTimeout, crypto: crypto.webcrypto };
box.window = box.self = box.global = box;
vm.createContext(box);
vm.runInContext(fs.readFileSync('../../../web/sodium.js', 'utf8'), box);
const frame = require('./assets/frame.js');
const input = JSON.parse(process.argv[1]);
box.sodium.ready.then(() => {
  process.stdout.write(JSON.stringify(input.passwords.map((pw) => frame.deriveAuthKey(box.sodium, pw, input.salt))));
});`
	input, err := json.Marshal(map[string]any{"salt": saltBase64, "passwords": passwords})
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(node, "-e", script, string(input)).Output()
	if err != nil {
		t.Fatalf("node failed: %v\n%s", err, out)
	}
	var keys []string
	if err := json.Unmarshal(out, &keys); err != nil || len(keys) != len(passwords) {
		t.Fatalf("node printed %q (%v), want %d keys", out, err, len(passwords))
	}
	return keys
}

// The page cannot sign in unless its JavaScript derives the same key the server expects, so this compares it with
// the Go reference above.
func TestPageDerivesTheSameAuthKeyAsTheReference(t *testing.T) {
	salt := []byte("0123456789abcdef")
	passwords := []string{"correct horse battery staple", "café ✓ with spaces ", "x"}
	got := pageDerive(t, base64.StdEncoding.EncodeToString(salt), passwords...)
	for i, password := range passwords {
		if want := referenceAuthKey(t, password, salt); got[i] != want {
			t.Errorf("password %q: page derived %s, reference is %s", password, got[i], want)
		}
	}
}
