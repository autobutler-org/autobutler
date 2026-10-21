// cspell:words smartctl findtime bantime maxretry journalmatch ignoreregex failregex

package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/hostnameutil"
	"github.com/autobutler-org/quark/pkg/util/sshutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

const (
	// sshdDropInDir is where Debian's sshd_config includes drop-ins from, at
	// the top of the file. sshd keeps the first value it reads for a keyword,
	// so drop-ins win over the main file, and among drop-ins the one sorting
	// first wins. The 00- prefix puts Quark's first.
	sshdDropInDir  = "/etc/ssh/sshd_config.d"
	sshdDropInPath = sshdDropInDir + "/00-quark.conf"

	// sshdDropInContent refuses root and everyone but quark, and points sshd
	// at the key file the service manages (#2131). Password login is allowed
	// for quark, and only works once an admin sets a password; keys always
	// work. MaxAuthTries and LoginGraceTime bound each connection, so a
	// guesser has to reconnect after three tries (#3578); they lock nothing
	// out, and a client with many keys can pick one with IdentitiesOnly. The
	// closing `Match all` ends the Match block, so nothing sshd reads after
	// this file lands inside it.
	sshdDropInContent = `# Written by quark install. Changes are overwritten on the next install.
PermitRootLogin no
AllowUsers ` + sshutil.LoginUser + `
MaxAuthTries 3
LoginGraceTime 30

Match User ` + sshutil.LoginUser + `
	AuthorizedKeysFile ` + sshutil.AuthorizedKeysPath + `
	PasswordAuthentication yes
Match all
`

	// fail2banDir is where fail2ban keeps its config. Quark never installs
	// fail2ban; when an admin has, install adds a jail for SSH passwords.
	fail2banDir      = "/etc/fail2ban"
	fail2banJailName = "quark-sshd"
	fail2banJailPath = fail2banDir + "/jail.d/" + fail2banJailName + ".local"

	// fail2banJailContent bans an address from port 22 only, for ten minutes,
	// after five failed passwords in ten minutes. The web app is unaffected.
	// Debian's sshd logs to the journal, not to a file.
	fail2banJailContent = `# Written by quark install. Changes are overwritten on the next install.
[` + fail2banJailName + `]
enabled = true
filter = ` + fail2banJailName + `
port = ssh
backend = systemd
maxretry = 5
findtime = 10m
bantime = 10m
`

	// fail2banFilterContent matches failed passwords and nothing else. The
	// stock sshd filter also counts offered keys, so a client trying the
	// wrong key first could get its admin banned. OpenSSH 9.8 and later logs
	// authentication from sshd-session rather than sshd.
	fail2banFilterContent = `# Written by quark install. Changes are overwritten on the next install.
[INCLUDES]
before = common.conf

[Definition]
_daemon = sshd(?:-session)?
failregex = ^%(__prefix_line)sFailed password for (?:invalid user )?\S+ from <HOST>(?: port \d+)?
ignoreregex =

[Init]
journalmatch = _SYSTEMD_UNIT=ssh.service
`

	// sshAccessHelperContent is the root helper behind sshutil.HelperPath: the
	// only root actions the service may take for SSH, each fixed. The one
	// sudoers entry for it takes any argument, so the script validates its
	// own. It never takes a username: the password it sets is always quark's,
	// and it reads it from stdin so it is never on a command line.
	sshAccessHelperContent = `#!/bin/sh
# Written by quark install. Changes are overwritten on the next install.
# Turns SSH access to this Quark on and off, and sets or clears the password
# of the ` + sshutil.LoginUser + ` login account.
set -eu
PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PATH

LOGIN_USER=` + sshutil.LoginUser + `

usage() {
	echo "usage: $0 enable|disable|set-password|clear-password" >&2
	exit 2
}

[ "$#" -eq 1 ] || usage

# Some Debian releases run sshd through ssh.socket. The socket and the service
# both want port 22, so the socket is kept off and the service does the work.
stop_socket() {
	if systemctl cat ssh.socket >/dev/null 2>&1; then
		systemctl disable --now ssh.socket
	fi
}

case "$1" in
enable)
	stop_socket
	systemctl enable --now ssh.service
	if command -v ufw >/dev/null 2>&1; then
		ufw allow 22/tcp
	fi
	;;
disable)
	stop_socket
	systemctl disable --now ssh.service
	if command -v ufw >/dev/null 2>&1; then
		# The rule may already be gone.
		ufw delete allow 22/tcp || true
	fi
	;;
set-password)
	password=""
	IFS= read -r password || true
	if [ -z "$password" ]; then
		echo "no password on stdin" >&2
		exit 2
	fi
	printf '%s:%s\n' "$LOGIN_USER" "$password" | chpasswd
	;;
clear-password)
	# '*' rather than the '!' that locking writes: no password matches it,
	# and the account stays unlocked so keys keep working.
	usermod -p '*' "$LOGIN_USER"
	;;
*)
	usage
	;;
esac
`
)

// sudoersContent is the whole of /etc/sudoers.d/quark: the mount rule, the
// SSH helper, the hostname helper, and the drive-health helper. The "" on
// the hostname entry lets sudo run that helper with no arguments only: the
// new name goes in on stdin. The drive-health entry takes the device path;
// the script accepts only a whole-disk name and then runs a fixed smartctl
// read, so the sudoers line does not list every disk.
func sudoersContent() string {
	mountsDir := filepath.Join(serviceDataDir, "data", "mounts")
	return fmt.Sprintf(
		"%s ALL=(root) NOPASSWD: /bin/mount * %s/*, /bin/umount %s/*\n"+
			"%s ALL=(root) NOPASSWD: %s\n"+
			"%s ALL=(root) NOPASSWD: %s \"\"\n"+
			"%s ALL=(root) NOPASSWD: %s\n",
		serviceUserName, mountsDir, mountsDir,
		serviceUserName, sshutil.HelperPath,
		serviceUserName, hostnameutil.HelperPath,
		serviceUserName, storageutil.HelperPath,
	)
}

// installRootHelper writes a helper root-owned, in a root-owned directory
// outside the self-updatable serviceBinDir, so the service can run it through
// sudo but never change it.
func installRootHelper(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dir, err)
	}
	if err := os.Chown(dir, 0, 0); err != nil {
		return fmt.Errorf("failed to set ownership on %s: %w", dir, err)
	}
	_, err := writeRootFileIfChanged(path, content, 0o755)
	return err
}

// installSSHDropIn writes the sshd drop-in when sshd is installed, and the
// fail2ban jail when fail2ban is too, and has whichever is running pick up
// what changed. Without sshd there is nothing to configure; the admin page
// says to install it.
func installSSHDropIn() error {
	if _, err := os.Stat(sshdDropInDir); err != nil {
		return nil
	}
	sshdChanged, err := writeRootFileIfChanged(sshdDropInPath, sshdDropInContent, 0o644)
	if err != nil {
		return err
	}
	jailChanged, err := installFail2banJail(fail2banDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return nil
	}
	// Each is a no-op when that service is not running.
	for unit, changed := range map[string]bool{"ssh.service": sshdChanged, "fail2ban.service": jailChanged} {
		if !changed {
			continue
		}
		if err := exec.Command("systemctl", "try-reload-or-restart", unit).Run(); err != nil {
			return fmt.Errorf("failed to reload %s: %w", unit, err)
		}
	}
	return nil
}

// installFail2banJail writes the SSH password jail and its filter under
// etcDir when fail2ban is installed there, and reports whether it wrote.
// Without fail2ban it writes nothing. The filter goes first, so the jail
// never names a filter that is not there.
func installFail2banJail(etcDir string) (bool, error) {
	files := []struct{ dir, name, content string }{
		{"filter.d", fail2banJailName + ".conf", fail2banFilterContent},
		{"jail.d", filepath.Base(fail2banJailPath), fail2banJailContent},
	}
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(etcDir, file.dir)); err != nil {
			return false, nil
		}
	}
	changed := false
	for _, file := range files {
		wrote, err := writeRootFileIfChanged(filepath.Join(etcDir, file.dir, file.name), file.content, 0o644)
		if err != nil {
			return changed, err
		}
		changed = changed || wrote
	}
	return changed, nil
}

// ensureLoginShell gives the service account a shell, which SSH needs to log
// it in. Accounts made before #2131 have nologin.
func ensureLoginShell() error {
	return exec.Command("usermod", "--shell", serviceLoginShell, serviceUserName).Run()
}

// lockRoot locks root's password (#2129). Armbian ships every image with root
// password 1234 and the image turns off the forced first-login change, so
// every Quark would otherwise share it. Root stays reachable through sudo.
// It checks first so the unit's start-time run does not rewrite /etc/shadow.
func lockRoot() error {
	out, err := exec.Command("passwd", "-S", "root").Output()
	if err != nil {
		return err
	}
	if passwordLocked(string(out)) {
		return nil
	}
	return exec.Command("passwd", "-l", "root").Run()
}

// passwordLocked reads the status field of a `passwd -S` line.
func passwordLocked(status string) bool {
	fields := strings.Fields(status)
	return len(fields) > 1 && fields[1] == "L"
}

// writeRootFileIfChanged writes path with writeRootFile unless it already holds
// content with mode, and reports whether it wrote. These are small files Quark
// writes itself, so reading one whole is fine.
func writeRootFileIfChanged(path, content string, mode os.FileMode) (bool, error) {
	if info, err := os.Stat(path); err == nil && info.Mode().Perm() == mode {
		if existing, err := os.ReadFile(path); err == nil && string(existing) == content {
			return false, nil
		}
	}
	return true, writeRootFile(path, content, mode)
}

// writeRootFile replaces path atomically with a root-owned file.
func writeRootFile(path, content string, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for %s: %w", path, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", path, err)
	}
	if err := os.Chown(tmp.Name(), 0, 0); err != nil {
		return fmt.Errorf("failed to set ownership on %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to replace %s: %w", path, err)
	}
	return nil
}
