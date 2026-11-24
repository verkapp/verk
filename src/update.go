package src

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	releasesAPIURL      = "https://api.github.com/repos/clementherve/todo/releases/latest"
	releasesPageURL     = "https://github.com/clementherve/todo/releases"
	updateCheckInterval = 6 * time.Hour
	updateCheckDelay    = 20 * time.Second
	maxUpdateBytes      = 512 << 20
)

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// releaseInfo is the result of a check. useGH is true when the repository is
// private and the metadata came from an authenticated gh CLI.
type releaseInfo struct {
	info     UpdateInfo
	asset    githubAsset
	checksum githubAsset
	tag      string
	useGH    bool
}

// updater keeps the last update check in memory so every screen can display
// the same state without hitting GitHub again.
type updater struct {
	mu            sync.Mutex
	info          UpdateInfo
	asset         githubAsset
	checksum      githubAsset
	tag           string
	useGH         bool
	stage         string
	checking      bool
	working       bool
	installCalled bool
}

// Version is overridden at build time with
// -ldflags "-X verk/src.Version=<tag>".
var Version = "1.4.0"

func (a *App) updateStatus() UpdateInfo {
	a.updates.mu.Lock()
	defer a.updates.mu.Unlock()
	return a.updates.info
}

func (a *App) emitUpdate() {
	a.emit("update:status", a.updateStatus())
}

// CheckForUpdates asks GitHub for the latest stable release.
func (a *App) CheckForUpdates() (UpdateInfo, error) {
	if err := a.ready(); err != nil {
		return UpdateInfo{}, err
	}
	return a.checkForUpdates(context.Background())
}

// UpdateStatus returns the cached update state, mostly used on first paint.
func (a *App) UpdateStatus() UpdateInfo {
	info := a.updateStatus()
	if info.CurrentVersion == "" {
		info.CurrentVersion = Version
	}
	return info
}

func (a *App) checkForUpdates(ctx context.Context) (UpdateInfo, error) {
	a.updates.mu.Lock()
	if a.updates.checking {
		info := a.updates.info
		a.updates.mu.Unlock()
		return info, nil
	}
	a.updates.checking = true
	a.updates.mu.Unlock()

	release, err := fetchLatestRelease(ctx)

	a.updates.mu.Lock()
	a.updates.checking = false
	if err == nil {
		sameVersion := a.updates.info.LatestVersion == release.info.LatestVersion
		if sameVersion && a.updates.info.Ready && a.updates.stage != "" {
			release.info.Ready = true
		}
		if sameVersion && a.updates.info.Downloading {
			release.info.Downloading = true
		}
		a.updates.info = release.info
		a.updates.asset = release.asset
		a.updates.checksum = release.checksum
		a.updates.tag = release.tag
		a.updates.useGH = release.useGH
		if !release.info.Ready && !release.info.Downloading {
			a.updates.stage = ""
		}
	}
	a.updates.mu.Unlock()
	a.emitUpdate()
	if err != nil {
		return UpdateInfo{CurrentVersion: Version}, err
	}
	return release.info, nil
}

func fetchLatestRelease(ctx context.Context) (releaseInfo, error) {
	release, err := fetchPublicRelease(ctx)
	useGH := false
	if err != nil {
		return releaseInfo{}, err
	}
	if release == nil {
		// Private (or not yet published) repository: fall back to the gh CLI,
		// which uses the user's existing GitHub credentials.
		private, ghErr := fetchReleaseWithGH(ctx)
		if ghErr != nil {
			return releaseInfo{
				info: UpdateInfo{
					CurrentVersion: Version,
					ReleaseURL:     releasesPageURL,
					Error:          "aucune release accessible ; rends le dépôt public ou connecte-toi avec `gh auth login`",
				},
			}, nil
		}
		release = private
		useGH = true
	}
	asset, checksum, ok := selectUpdateAsset(release.Assets)
	info := UpdateInfo{
		CurrentVersion: Version,
		LatestVersion:  strings.TrimPrefix(release.TagName, "v"),
		Available:      compareVersions(release.TagName, Version) > 0 && ok,
		Notes:          clip(strings.TrimSpace(release.Body), 4000),
		ReleaseURL:     release.HTMLURL,
		PublishedAt:    release.PublishedAt.UnixMilli(),
	}
	if info.ReleaseURL == "" {
		info.ReleaseURL = releasesPageURL
	}
	if compareVersions(release.TagName, Version) > 0 && !ok {
		info.Error = "Aucune archive ne correspond à cet ordinateur."
	}
	return releaseInfo{info: info, asset: asset, checksum: checksum, tag: release.TagName, useGH: useGH}, nil
}

// fetchPublicRelease returns nil, nil when the repository is not publicly
// visible, so the caller can retry with gh.
func fetchPublicRelease(ctx context.Context) (*githubRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Verk/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("impossible de joindre GitHub : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub a répondu %s", resp.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&release); err != nil {
		return nil, fmt.Errorf("réponse de GitHub illisible : %w", err)
	}
	return &release, nil
}

func fetchReleaseWithGH(ctx context.Context) (*githubRelease, error) {
	gh, err := resolveGHPath()
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, gh, "api", "repos/clementherve/todo/releases/latest")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var release githubRelease
	if err := json.Unmarshal(out, &release); err != nil {
		return nil, err
	}
	return &release, nil
}

// resolveGHPath avoids relying on PATH: a macOS app launched from Finder
// inherits a minimal environment and would miss Homebrew's gh.
func resolveGHPath() (string, error) {
	if path, err := exec.LookPath("gh"); err == nil {
		return path, nil
	}
	home, _ := os.UserHomeDir()
	var candidates []string
	if runtime.GOOS == "windows" {
		if base := os.Getenv("ProgramFiles"); base != "" {
			candidates = append(candidates, filepath.Join(base, "GitHub CLI", "gh.exe"))
		}
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			candidates = append(candidates, filepath.Join(base, "Programs", "GitHub CLI", "gh.exe"))
		}
	} else {
		if runtime.GOOS == "darwin" {
			candidates = append(candidates, "/opt/homebrew/bin/gh", "/usr/local/bin/gh")
		}
		if home != "" {
			candidates = append(candidates, filepath.Join(home, ".local", "bin", "gh"))
		}
		candidates = append(candidates, "/usr/bin/gh", "/bin/gh")
	}
	for _, candidate := range candidates {
		if isExecutableFile(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("gh est introuvable")
}

// downloadAsset either streams the public asset or asks gh to fetch it from
// the private repository.
func downloadAsset(ctx context.Context, asset githubAsset, tag string, useGH bool, destination string) error {
	if !useGH {
		return downloadFile(ctx, asset.BrowserDownloadURL, destination, maxUpdateBytes)
	}
	gh, err := resolveGHPath()
	if err != nil {
		return errors.New("gh est introuvable pour télécharger la mise à jour")
	}
	directory := filepath.Dir(destination)
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(runCtx, gh, "release", "download", tag,
		"--repo", "clementherve/todo",
		"--pattern", asset.Name,
		"--dir", directory,
		"--clobber")
	hideWindow(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("téléchargement de la mise à jour : %s", clip(strings.TrimSpace(string(output)), 300))
	}
	downloaded := filepath.Join(directory, asset.Name)
	if downloaded != destination {
		if err := os.Rename(downloaded, destination); err != nil {
			return fmt.Errorf("téléchargement de la mise à jour : %w", err)
		}
	}
	return nil
}

func selectUpdateAsset(assets []githubAsset) (githubAsset, githubAsset, bool) {
	want := ""
	switch runtime.GOOS {
	case "darwin":
		want = "macos"
	case "windows":
		want = "windows"
	default:
		return githubAsset{}, githubAsset{}, false
	}
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if !strings.HasSuffix(name, ".zip") || !strings.Contains(name, want) {
			continue
		}
		for _, candidate := range assets {
			if strings.EqualFold(candidate.Name, asset.Name+".sha256") {
				return asset, candidate, true
			}
		}
		return asset, githubAsset{}, true
	}
	return githubAsset{}, githubAsset{}, false
}

// DownloadUpdate downloads and unpacks the release in the application data
// folder. It does not touch the installed app yet.
func (a *App) DownloadUpdate() (UpdateInfo, error) {
	if err := a.ready(); err != nil {
		return UpdateInfo{}, err
	}
	a.updates.mu.Lock()
	asset := a.updates.asset
	checksum := a.updates.checksum
	tag := a.updates.tag
	useGH := a.updates.useGH
	info := a.updates.info
	a.updates.mu.Unlock()
	if !info.Available {
		return info, errors.New("aucune mise à jour à télécharger")
	}
	if info.Ready {
		return info, nil
	}
	if asset.BrowserDownloadURL == "" {
		return info, errors.New("aucune archive disponible pour cet ordinateur")
	}

	a.updates.mu.Lock()
	if a.updates.working {
		a.updates.mu.Unlock()
		return info, errors.New("téléchargement déjà en cours")
	}
	a.updates.working = true
	a.updates.info.Downloading = true
	a.updates.info.Error = ""
	a.updates.mu.Unlock()
	a.emitUpdate()

	stage, err := a.downloadAndStage(context.Background(), info.LatestVersion, asset, checksum, tag, useGH)

	a.updates.mu.Lock()
	a.updates.working = false
	a.updates.info.Downloading = false
	if err != nil {
		a.updates.info.Error = err.Error()
	} else {
		a.updates.stage = stage
		a.updates.info.Ready = true
	}
	result := a.updates.info
	a.updates.mu.Unlock()
	a.emitUpdate()
	return result, err
}

func (a *App) downloadAndStage(ctx context.Context, version string, asset, checksum githubAsset, tag string, useGH bool) (string, error) {
	staging := filepath.Join(a.store.dataDir, "updates", safeVersion(version))
	if err := os.RemoveAll(staging); err != nil {
		return "", fmt.Errorf("préparation du dossier de mise à jour : %w", err)
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return "", fmt.Errorf("préparation du dossier de mise à jour : %w", err)
	}
	zipPath := filepath.Join(staging, "update.zip")
	if err := downloadAsset(ctx, asset, tag, useGH, zipPath); err != nil {
		return "", err
	}
	if err := verifyDownload(ctx, checksum, zipPath, tag, useGH); err != nil {
		return "", err
	}
	return extractUpdate(zipPath, staging)
}

func downloadFile(ctx context.Context, url, destination string, limit int64) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Verk/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("téléchargement de la mise à jour : %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("téléchargement de la mise à jour : GitHub a répondu %s", resp.Status)
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("écriture de la mise à jour : %w", err)
	}
	written, copyErr := io.Copy(out, io.LimitReader(resp.Body, limit+1))
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("téléchargement de la mise à jour : %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("écriture de la mise à jour : %w", closeErr)
	}
	if written > limit {
		return errors.New("l'archive de mise à jour est trop volumineuse")
	}
	return nil
}

// verifyDownload checks the optional .sha256 asset published next to the
// archive. Older releases without a checksum are still accepted.
func verifyDownload(ctx context.Context, checksum githubAsset, path, tag string, useGH bool) error {
	if checksum.BrowserDownloadURL == "" && checksum.Name == "" {
		return nil
	}
	checksumPath := path + ".sha256"
	if err := downloadAsset(ctx, checksum, tag, useGH, checksumPath); err != nil {
		return err
	}
	content, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("lecture de l'empreinte de mise à jour : %w", err)
	}
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return errors.New("empreinte de mise à jour illisible")
	}
	expected := strings.ToLower(strings.TrimPrefix(fields[0], "sha256:"))
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("lecture de l'archive de mise à jour : %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return errors.New("l'archive de mise à jour ne correspond pas à son empreinte")
	}
	return nil
}

func extractUpdate(zipPath, staging string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("archive de mise à jour illisible : %w", err)
	}
	defer func() { _ = reader.Close() }()

	target := ""
	switch runtime.GOOS {
	case "darwin":
		target = firstAppBundle(reader.File)
		if target == "" {
			return "", errors.New("l'archive ne contient pas d'application Verk")
		}
	case "windows":
		target = "Todo.exe"
	default:
		return "", errors.New("les mises à jour automatiques ne sont pas disponibles sur ce système")
	}

	destination := filepath.Join(staging, filepath.FromSlash(target))
	for _, entry := range reader.File {
		if err := extractZipEntry(entry, staging, target); err != nil {
			return "", err
		}
	}
	if runtime.GOOS == "darwin" {
		destination = filepath.Join(staging, filepath.FromSlash(strings.Split(target, "/")[0]))
	}
	if info, err := os.Stat(destination); err != nil || (!info.IsDir() && info.Size() == 0) {
		return "", errors.New("l'archive de mise à jour est incomplète")
	}
	return destination, nil
}

func firstAppBundle(files []*zip.File) string {
	for _, file := range files {
		parts := strings.Split(file.Name, "/")
		for _, part := range parts {
			if strings.HasSuffix(part, ".app") {
				index := strings.Index(file.Name, part)
				return file.Name[:index+len(part)]
			}
		}
	}
	return ""
}

func extractZipEntry(entry *zip.File, staging, target string) error {
	name := filepath.ToSlash(entry.Name)
	prefix := target
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if name != target && !strings.HasPrefix(name, prefix) {
		return nil
	}
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if cleaned == "." || strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return errors.New("archive de mise à jour invalide")
	}
	destination := filepath.Join(staging, cleaned)
	if entry.FileInfo().IsDir() {
		return os.MkdirAll(destination, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	mode := entry.Mode()
	source, err := entry.Open()
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	if mode&os.ModeSymlink != 0 {
		link, err := io.ReadAll(io.LimitReader(source, 4096))
		if err != nil {
			return err
		}
		_ = os.Remove(destination)
		return os.Symlink(string(link), destination)
	}
	if mode&0o111 == 0 {
		mode = 0o644
	} else {
		mode = 0o755
	}
	targetFile, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(targetFile, io.LimitReader(source, maxUpdateBytes)); err != nil {
		_ = targetFile.Close()
		return err
	}
	return targetFile.Close()
}

// InstallUpdate swaps the running application with the staged one and
// relaunches it.
func (a *App) InstallUpdate() error {
	if err := a.ready(); err != nil {
		return err
	}
	a.updates.mu.Lock()
	stage := a.updates.stage
	ready := a.updates.info.Ready
	a.updates.mu.Unlock()
	if stage == "" || !ready {
		return errors.New("aucune mise à jour n'est prête")
	}
	if err := a.launchInstaller(stage, true); err != nil {
		return err
	}
	a.quitApp()
	return nil
}

// OpenUpdatePage is the manual fallback when the automatic install fails.
func (a *App) OpenUpdatePage() {
	target := a.updateStatus().ReleaseURL
	if target == "" {
		target = releasesPageURL
	}
	if a.wapp != nil {
		_ = a.wapp.Browser.OpenURL(target)
	}
}

func (a *App) launchInstaller(stage string, relaunch bool) error {
	a.updates.mu.Lock()
	if a.updates.installCalled {
		a.updates.mu.Unlock()
		return nil
	}
	a.updates.installCalled = true
	a.updates.mu.Unlock()

	executable, err := os.Executable()
	if err != nil {
		a.resetInstaller()
		return fmt.Errorf("localisation de l'application : %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		a.resetInstaller()
		return fmt.Errorf("localisation de l'application : %w", err)
	}
	if runtime.GOOS == "darwin" {
		if err := launchMacInstaller(executable, stage, relaunch); err != nil {
			a.resetInstaller()
			return err
		}
		return nil
	}
	if runtime.GOOS == "windows" {
		if err := launchWindowsInstaller(executable, stage, relaunch); err != nil {
			a.resetInstaller()
			return err
		}
		return nil
	}
	a.resetInstaller()
	return errors.New("installation automatique non prise en charge sur ce système")
}

func (a *App) resetInstaller() {
	a.updates.mu.Lock()
	a.updates.installCalled = false
	a.updates.mu.Unlock()
}

func launchMacInstaller(executable, stage string, relaunch bool) error {
	bundle := macBundlePath(executable)
	if bundle == "" {
		return errors.New("l'application ne tourne pas depuis un bundle .app")
	}
	if strings.Contains(bundle, "/AppTranslocation/") {
		return errors.New("déplace Verk dans le dossier Applications, puis relance la mise à jour")
	}
	if !canWriteDirectory(filepath.Dir(bundle)) {
		return errors.New("le dossier Applications n'est pas accessible en écriture")
	}
	script := macInstallScript(bundle, stage, os.Getpid(), relaunch)
	path := filepath.Join(filepath.Dir(filepath.Dir(stage)), "install.sh")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return fmt.Errorf("préparation de l'installation : %w", err)
	}
	cmd := exec.Command("/bin/sh", path)
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("lancement de l'installation : %w", err)
	}
	return nil
}

func macBundlePath(executable string) string {
	current := executable
	for i := 0; i < 6; i++ {
		if strings.HasSuffix(current, ".app") {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func macInstallScript(bundle, stage string, pid int, relaunch bool) string {
	relaunchLine := ""
	if relaunch {
		relaunchLine = "open " + shellQuote(bundle) + "\n"
	}
	return "#!/bin/sh\n" +
		"while kill -0 " + strconv.Itoa(pid) + " 2>/dev/null; do sleep 0.3; done\n" +
		"rm -rf " + shellQuote(bundle) + "\n" +
		"mv " + shellQuote(stage) + " " + shellQuote(bundle) + "\n" +
		"xattr -dr com.apple.quarantine " + shellQuote(bundle) + " 2>/dev/null || true\n" +
		relaunchLine +
		"rm -rf " + shellQuote(filepath.Dir(stage)) + "\n" +
		"rm -f \"$0\"\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func launchWindowsInstaller(executable, stage string, relaunch bool) error {
	script := windowsInstallScript(executable, stage, os.Getpid(), relaunch)
	path := filepath.Join(filepath.Dir(filepath.Dir(stage)), "install.bat")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return fmt.Errorf("préparation de l'installation : %w", err)
	}
	cmd := exec.Command("cmd", "/C", path)
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("lancement de l'installation : %w", err)
	}
	return nil
}

func windowsInstallScript(executable, stage string, pid int, relaunch bool) string {
	relaunchLine := ""
	if relaunch {
		relaunchLine = "start \"\" \"" + batchPath(executable) + "\"\r\n"
	}
	return "@echo off\r\n" +
		"setlocal\r\n" +
		":wait\r\n" +
		"tasklist /FI \"PID eq " + strconv.Itoa(pid) + "\" 2>NUL | find \"" + strconv.Itoa(pid) + "\" >NUL\r\n" +
		"if not errorlevel 1 (\r\n" +
		"  timeout /t 1 /nobreak >NUL\r\n" +
		"  goto wait\r\n" +
		")\r\n" +
		"move /Y \"" + batchPath(stage) + "\" \"" + batchPath(executable) + "\" >NUL\r\n" +
		"if errorlevel 1 exit /b 1\r\n" +
		relaunchLine +
		"rmdir /S /Q \"" + batchPath(filepath.Dir(stage)) + "\" >NUL 2>&1\r\n" +
		"del \"%~f0\"\r\n"
}

func batchPath(value string) string {
	return strings.ReplaceAll(value, "%", "%%")
}

func canWriteDirectory(dir string) bool {
	probe, err := os.CreateTemp(dir, ".verk-write-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return true
}

// autoUpdateLoop checks quietly at startup and every six hours. When a newer
// release is found it downloads it in the background so the update is ready
// without the user opening a browser.
func (a *App) autoUpdateLoop() {
	select {
	case <-time.After(updateCheckDelay):
	case <-a.updateStop:
		return
	}
	for {
		settings, err := a.store.getSettings()
		if err == nil && settings.AutoUpdate {
			info, checkErr := a.checkForUpdates(context.Background())
			if checkErr == nil && info.Available {
				_, _ = a.DownloadUpdate()
			}
		}
		select {
		case <-time.After(updateCheckInterval):
		case <-a.updateStop:
			return
		}
	}
}

// installUpdateOnQuit applies a downloaded update when the user closes the app,
// so the next launch runs the new version.
func (a *App) installUpdateOnQuit() {
	settings, err := a.store.getSettings()
	if err != nil || !settings.AutoUpdate {
		return
	}
	a.updates.mu.Lock()
	stage := a.updates.stage
	info := a.updates.info
	a.updates.mu.Unlock()
	if stage == "" || !info.Ready {
		return
	}
	if err := a.launchInstaller(stage, false); err != nil {
		// Keep the app usable: the update stays staged and the user can
		// restart it from the settings later.
		a.updates.mu.Lock()
		a.updates.installCalled = false
		a.updates.info.Error = err.Error()
		a.updates.mu.Unlock()
	}
}

// compareVersions implements enough of semantic versioning to compare tags
// such as v1.2.3, 1.2.3-dev.4 and 1.2.4. It returns -1, 0 or 1.
func compareVersions(left, right string) int {
	a, aPre := splitVersion(left)
	b, bPre := splitVersion(right)
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	if aPre == "" && bPre == "" {
		return 0
	}
	if aPre == "" {
		return 1
	}
	if bPre == "" {
		return -1
	}
	return comparePrerelease(aPre, bPre)
}

func splitVersion(value string) ([3]int, string) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	if index := strings.IndexByte(value, '+'); index >= 0 {
		value = value[:index]
	}
	prerelease := ""
	if index := strings.IndexByte(value, '-'); index >= 0 {
		prerelease = value[index+1:]
		value = value[:index]
	}
	var numbers [3]int
	for index, part := range strings.Split(value, ".") {
		if index >= 3 {
			break
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			number = 0
		}
		numbers[index] = number
	}
	return numbers, prerelease
}

func comparePrerelease(left, right string) int {
	a := strings.Split(left, ".")
	b := strings.Split(right, ".")
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		aNumber, aErr := strconv.Atoi(a[i])
		bNumber, bErr := strconv.Atoi(b[i])
		switch {
		case aErr == nil && bErr == nil:
			if aNumber < bNumber {
				return -1
			}
			return 1
		case aErr == nil:
			return -1
		case bErr == nil:
			return 1
		default:
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	default:
		return 0
	}
}

func safeVersion(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "latest"
	}
	return builder.String()
}
