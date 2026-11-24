package src

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const syncDirName = "sync"

// snapshotFileName held the whole dataset up to format 2 (Verk 2.0, Todo).
// Since format 3 it only holds the fence that stops those versions; the data
// lives in profiles/ (see sync_layout.go). It keeps its pre-Verk name because
// older devices look for it.
const snapshotFileName = "todo.json"

// resolveGitPath mirrors the Claude Code lookup: a macOS app launched from the
// Finder inherits a minimal PATH, so Homebrew git is often invisible.
func resolveGitPath() (string, error) {
	if path, err := exec.LookPath("git"); err == nil {
		return path, nil
	}
	home, _ := os.UserHomeDir()
	var candidates []string
	if runtime.GOOS == "windows" {
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if base != "" {
				candidates = append(candidates, filepath.Join(base, "Git", "cmd", "git.exe"))
			}
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			candidates = append(candidates, filepath.Join(local, "Programs", "Git", "cmd", "git.exe"))
		}
		if home != "" {
			candidates = append(candidates, filepath.Join(home, "scoop", "shims", "git.exe"))
		}
	} else {
		if runtime.GOOS == "darwin" {
			candidates = append(candidates, "/opt/homebrew/bin/git", "/usr/local/bin/git")
		}
		candidates = append(candidates, "/usr/bin/git", "/bin/git")
	}
	for _, candidate := range candidates {
		if isExecutableFile(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("git est introuvable sur cette machine")
}

// gitInstalled reports whether the git CLI is available. Sync shells out to git
// so the app keeps using the machine's existing credentials and SSH setup.
func gitInstalled() bool {
	_, err := resolveGitPath()
	return err == nil
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	git, err := resolveGitPath()
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, git, args...)
	cmd.Dir = dir
	// Never let git open an interactive credential prompt from a GUI app: it
	// would hang until the timeout with no visible terminal.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	hideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("git met trop de temps à répondre")
	}
	if runErr != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = runErr.Error()
		}
		return "", errors.New(clip(detail, 400))
	}
	return stdout.String(), nil
}

func gitRefExists(ctx context.Context, dir, ref string) bool {
	_, err := runGit(ctx, dir, "show-ref", "--verify", "--quiet", ref)
	return err == nil
}

// detectRemoteBranch asks the remote for its default branch. An empty remote
// has no HEAD, in which case the app starts on main.
func detectRemoteBranch(ctx context.Context, dir, remote string) (string, error) {
	out, err := runGit(ctx, dir, "ls-remote", "--symref", remote, "HEAD")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "ref: refs/heads/") {
			continue
		}
		name := strings.TrimPrefix(line, "ref: refs/heads/")
		if fields := strings.Fields(name); len(fields) > 0 && fields[0] != "" {
			return fields[0], nil
		}
	}
	return "main", nil
}

// ensureSyncRepo prepares the app-managed checkout. It never touches a folder
// outside the application data directory.
func ensureSyncRepo(ctx context.Context, dir, remote, branch string) error {
	if branch == "" {
		branch = "main"
	}
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		return err
	}
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); errors.Is(err, os.ErrNotExist) {
		if _, err := runGit(ctx, dir, "init", "-b", branch); err != nil {
			// git < 2.28 has no -b; fall back to a rename.
			if _, initErr := runGit(ctx, dir, "init"); initErr != nil {
				return initErr
			}
			if _, checkoutErr := runGit(ctx, dir, "checkout", "-B", branch); checkoutErr != nil {
				return checkoutErr
			}
		}
	} else if err != nil {
		return err
	}

	url, urlErr := runGit(ctx, dir, "remote", "get-url", "origin")
	if urlErr != nil || strings.TrimSpace(url) != remote {
		if _, err := runGit(ctx, dir, "remote", "add", "origin", remote); err != nil {
			if _, err := runGit(ctx, dir, "remote", "set-url", "origin", remote); err != nil {
				return err
			}
		}
	}
	if _, err := runGit(ctx, dir, "checkout", "-B", branch); err != nil {
		return err
	}
	// The checkout belongs to the app, not to the person's git identity.
	// Repo-local values keep commits working even with no global configuration.
	if _, err := runGit(ctx, dir, "config", "user.name", "Verk"); err != nil {
		return err
	}
	if _, err := runGit(ctx, dir, "config", "user.email", "verk@localhost"); err != nil {
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}

// copyMissingImages copies image files between the local image directory and
// the git checkout, skipping files already present at the destination. Images
// are immutable once imported, so a name collision always has the same bytes.
func copyMissingImages(images []snapshotImage, srcDir, dstDir string) error {
	for _, img := range images {
		if img.File == "" {
			continue
		}
		dst := filepath.Join(dstDir, img.File)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		src := filepath.Join(srcDir, img.File)
		if err := copyFile(src, dst); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("copie de l'image %s : %w", img.File, err)
		}
	}
	return nil
}
