// cspell:words nmaxretry publickey

package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/sshutil"
)

// The mount rule must not move, and SSH adds exactly one exact-path entry: no
// broad systemctl, ufw or chpasswd rights (#2131). Renaming adds one more,
// which sudo runs with no arguments at all: no broad hostnamectl (#2344).
// Drive health adds one more. The script, not sudoers, checks the device path.
func TestSudoersContent(t *testing.T) {
	want := "quark ALL=(root) NOPASSWD: /bin/mount * /var/lib/quark/data/mounts/*, /bin/umount /var/lib/quark/data/mounts/*\n" +
		"quark ALL=(root) NOPASSWD: /usr/local/libexec/quark/ssh-access\n" +
		"quark ALL=(root) NOPASSWD: /usr/local/libexec/quark/set-hostname \"\"\n" +
		"quark ALL=(root) NOPASSWD: /usr/local/libexec/quark/drive-health\n"
	if got := sudoersContent(); got != want {
		t.Errorf("sudoers content:\n%s\nwant:\n%s", got, want)
	}
	rule := regexp.MustCompile(`^quark ALL=\(root\) NOPASSWD: /\S`)
	for line := range strings.SplitSeq(strings.TrimSuffix(sudoersContent(), "\n"), "\n") {
		if !rule.MatchString(line) {
			t.Errorf("sudoers line %q is not a quark NOPASSWD rule for an absolute path", line)
		}
	}
}

// Root runs the helper, so the service must not be able to rewrite it: it
// cannot live in the directory self-update writes to.
func TestSSHHelperOutsideSelfUpdatableDir(t *testing.T) {
	if !filepath.IsAbs(sshutil.HelperPath) {
		t.Fatalf("helper path %s is not absolute", sshutil.HelperPath)
	}
	if strings.HasPrefix(sshutil.HelperPath, serviceBinDir+"/") || strings.HasPrefix(sshutil.HelperPath, serviceDataDir+"/") {
		t.Errorf("helper %s sits somewhere the service can write", sshutil.HelperPath)
	}
}

func TestSSHAccessHelperScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ssh-access")
	if err := os.WriteFile(path, []byte(sshAccessHelperContent), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("sh", "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("sh -n: %v: %s", err, out)
	}
	// Bad arguments exit before anything privileged runs.
	for _, args := range [][]string{{}, {"reboot"}, {"enable", "extra"}} {
		cmd := exec.Command(path, args...)
		if err := cmd.Run(); err == nil || cmd.ProcessState.ExitCode() != 2 {
			t.Errorf("ssh-access %v = %v, want exit 2", args, err)
		}
	}
	for _, want := range []string{"set -eu", "LOGIN_USER=quark", "| chpasswd", "usermod -p '*'", "ufw allow 22/tcp", "systemctl enable --now ssh.service"} {
		if !strings.Contains(sshAccessHelperContent, want) {
			t.Errorf("helper is missing %q", want)
		}
	}
	if strings.Contains(sshAccessHelperContent, "passwd -l") {
		t.Error("helper locks the account with passwd -l, which also blocks key logins")
	}
}

func TestSSHDropIn(t *testing.T) {
	for _, want := range []string{
		"\nPermitRootLogin no\n",
		"\nAllowUsers quark\n",
		"\nMatch User quark\n",
		"AuthorizedKeysFile " + sshutil.AuthorizedKeysPath + "\n",
	} {
		if !strings.Contains(sshdDropInContent, want) {
			t.Errorf("drop-in is missing %q:\n%s", want, sshdDropInContent)
		}
	}
	// Both limits are global keywords and must sit before the first Match, or
	// sshd rejects the file (LoginGraceTime) or scopes them to one block.
	global := sshdDropInContent[:strings.Index(sshdDropInContent, "\nMatch ")]
	for _, want := range []string{"\nMaxAuthTries 3\n", "\nLoginGraceTime 30\n"} {
		if !strings.Contains(global, want) {
			t.Errorf("drop-in is missing global %q before its Match block:\n%s", want, sshdDropInContent)
		}
	}
	if !strings.HasSuffix(sshdDropInContent, "Match all\n") {
		t.Error("drop-in must close its Match block so later config does not land in it")
	}
	if !strings.HasPrefix(sshdDropInPath, sshdDropInDir+"/") || !strings.HasSuffix(sshdDropInPath, ".conf") {
		t.Errorf("sshd includes only *.conf from %s, not %s", sshdDropInDir, sshdDropInPath)
	}
}

// The jail bans an address only for SSH, and only for failed passwords: a
// client offering the wrong keys before the right one is never banned (#3578).
func TestFail2banJail(t *testing.T) {
	for _, want := range []string{
		"\n[" + fail2banJailName + "]\n",
		"\nenabled = true\n",
		"\nfilter = " + fail2banJailName + "\n",
		"\nport = ssh\n",
		"\nbackend = systemd\n",
		"\nmaxretry = ",
	} {
		if !strings.Contains(fail2banJailContent, want) {
			t.Errorf("jail is missing %q:\n%s", want, fail2banJailContent)
		}
	}
	if !strings.Contains(fail2banFilterContent, "Failed password for") {
		t.Errorf("filter does not match failed passwords:\n%s", fail2banFilterContent)
	}
	if strings.Contains(strings.ToLower(fail2banFilterContent), "publickey") {
		t.Errorf("filter matches key failures, which could ban a key-only admin:\n%s", fail2banFilterContent)
	}
	if !strings.HasSuffix(fail2banJailPath, ".local") && !strings.HasSuffix(fail2banJailPath, ".conf") {
		t.Errorf("fail2ban reads only *.conf and *.local from jail.d, not %s", fail2banJailPath)
	}
}

// Quark never installs fail2ban, so without it no fail2ban config is written.
func TestInstallFail2banJail_SkipsWithoutFail2ban(t *testing.T) {
	etc := t.TempDir()
	changed, err := installFail2banJail(etc)
	if err != nil {
		t.Fatalf("installFail2banJail: %v", err)
	}
	if changed {
		t.Error("reported a write with no fail2ban on the host")
	}
	entries, err := os.ReadDir(etc)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("wrote %v with no fail2ban on the host", entries)
	}
}

func TestInstallFail2banJail_WritesWithFail2ban(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("writeRootFile gives the file to root, which needs root")
	}
	etc := t.TempDir()
	for _, dir := range []string{"jail.d", "filter.d"} {
		if err := os.Mkdir(filepath.Join(etc, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for i, wantChanged := range []bool{true, false} {
		changed, err := installFail2banJail(etc)
		if err != nil {
			t.Fatalf("run %d: installFail2banJail: %v", i, err)
		}
		if changed != wantChanged {
			t.Errorf("run %d: changed = %v, want %v", i, changed, wantChanged)
		}
	}
	jail, _ := os.ReadFile(filepath.Join(etc, "jail.d", filepath.Base(fail2banJailPath)))
	filter, _ := os.ReadFile(filepath.Join(etc, "filter.d", fail2banJailName+".conf"))
	if string(jail) != fail2banJailContent || string(filter) != fail2banFilterContent {
		t.Errorf("jail.d or filter.d holds the wrong content:\n%s\n%s", jail, filter)
	}
}

// --system-only runs on every start, so a current file must not be rewritten
// (#2120).
func TestWriteRootFileIfChanged_LeavesACurrentFileAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quark")
	if err := os.WriteFile(path, []byte("current\n"), 0o440); err != nil {
		t.Fatal(err)
	}
	old := time.Unix(1_000_000_000, 0)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	changed, err := writeRootFileIfChanged(path, "current\n", 0o440)
	if err != nil {
		t.Fatalf("writeRootFileIfChanged: %v", err)
	}
	if changed {
		t.Error("reported a write for a file that was already current")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) {
		t.Errorf("file was rewritten: mtime %v, want %v", info.ModTime(), old)
	}
}

func TestWriteRootFileIfChanged_RewritesAStaleFile(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("writeRootFile gives the file to root, which needs root")
	}
	path := filepath.Join(t.TempDir(), "quark")
	if err := os.WriteFile(path, []byte("stale\n"), 0o440); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []os.FileMode{0o440, 0o400} {
		changed, err := writeRootFileIfChanged(path, "current\n", mode)
		if err != nil {
			t.Fatalf("writeRootFileIfChanged: %v", err)
		}
		if !changed {
			t.Errorf("mode %o: reported no write for a stale file", mode)
		}
	}
	if got, _ := os.ReadFile(path); string(got) != "current\n" {
		t.Errorf("file holds %q", got)
	}
}

func TestPasswordLocked(t *testing.T) {
	for status, want := range map[string]bool{
		"root L 2026-01-01 0 99999 7 -1\n":  true,
		"root P 2026-01-01 0 99999 7 -1\n":  false,
		"root NP 2026-01-01 0 99999 7 -1\n": false,
		"":                                  false,
	} {
		if got := passwordLocked(status); got != want {
			t.Errorf("passwordLocked(%q) = %v, want %v", status, got, want)
		}
	}
}
