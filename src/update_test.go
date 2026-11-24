package src

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{"v1.2.3", "1.2.2", 1},
		{"1.2.2", "v1.2.2", 0},
		{"1.2.2", "1.2.10", -1},
		{"1.2.2-dev.1", "1.2.2", -1},
		{"1.2.2", "1.2.2-dev.1", 1},
		{"1.2.2-dev.2", "1.2.2-dev.1", 1},
		{"1.3.0-dev.1", "1.2.9", 1},
		{"2.0", "1.9.9", 1},
		{"v1.2.3+build.7", "1.2.3", 0},
	}
	for _, test := range cases {
		if got := compareVersions(test.left, test.right); got != test.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestSelectUpdateAsset(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("no automatic update asset on this platform")
	}
	assets := []githubAsset{
		{Name: "Verk-macos-universal.zip", BrowserDownloadURL: "https://example.test/macos.zip"},
		{Name: "Verk-macos-universal.zip.sha256", BrowserDownloadURL: "https://example.test/macos.zip.sha256"},
		{Name: "Verk-windows-x64.zip", BrowserDownloadURL: "https://example.test/windows.zip"},
		{Name: "Verk-windows-x64.zip.sha256", BrowserDownloadURL: "https://example.test/windows.zip.sha256"},
	}
	asset, checksum, ok := selectUpdateAsset(assets)
	if !ok {
		t.Fatal("expected an asset for this platform")
	}
	if runtime.GOOS == "darwin" && asset.Name != "Verk-macos-universal.zip" {
		t.Fatalf("unexpected macOS asset: %q", asset.Name)
	}
	if runtime.GOOS == "windows" && asset.Name != "Verk-windows-x64.zip" {
		t.Fatalf("unexpected Windows asset: %q", asset.Name)
	}
	if checksum.Name != asset.Name+".sha256" {
		t.Fatalf("checksum asset should match the archive, got %q", checksum.Name)
	}
}

func TestVerifyDownloadChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.zip")
	content := []byte("verk update archive")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	expected := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(expected + "  archive.zip\n"))
	}))
	defer server.Close()

	checksum := githubAsset{Name: "archive.zip.sha256", BrowserDownloadURL: server.URL}
	if err := verifyDownload(t.Context(), checksum, path, "", false); err != nil {
		t.Fatalf("verifyDownload: %v", err)
	}

	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("deadbeef  archive.zip\n"))
	}))
	defer badServer.Close()
	if err := verifyDownload(t.Context(), githubAsset{BrowserDownloadURL: badServer.URL}, path, "", false); err == nil {
		t.Fatal("a wrong checksum should be rejected")
	}
}

func writeTestAppZip(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entries := map[string]struct {
		content string
		mode    os.FileMode
	}{
		"Verk.app/":                      {"", 0o755 | os.ModeDir},
		"Verk.app/Contents/":             {"", 0o755 | os.ModeDir},
		"Verk.app/Contents/MacOS/":       {"", 0o755 | os.ModeDir},
		"Verk.app/Contents/MacOS/Verk":   {"binary", 0o755},
		"Verk.app/Contents/Info.plist":   {"plist", 0o644},
		"Verk.app/Contents/evil/../../x": {"escape", 0o644},
		"Another/ignored.txt":            {"ignored", 0o644},
	}
	for name, entry := range entries {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(entry.mode)
		part, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(entry.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractUpdate(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS app extraction")
	}
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "update.zip")
	writeTestAppZip(t, zipPath)

	staging := filepath.Join(dir, "stage")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	app, err := extractUpdate(zipPath, staging)
	if err != nil {
		t.Fatalf("extractUpdate: %v", err)
	}
	binary := filepath.Join(app, "Contents", "MacOS", "Verk")
	info, err := os.Stat(binary)
	if err != nil {
		t.Fatalf("extracted binary: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatal("the app binary should stay executable")
	}
	if _, err := os.Stat(filepath.Join(staging, "Another")); !os.IsNotExist(err) {
		t.Fatal("entries outside the app bundle should be ignored")
	}
}

func TestDownloadAndStage(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS app extraction")
	}
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "Verk-macos-universal.zip")
	writeTestAppZip(t, zipPath)
	content, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	checksumText := hex.EncodeToString(sum[:]) + "  Verk-macos-universal.zip\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/update.zip" {
			_, _ = w.Write(content)
			return
		}
		_, _ = w.Write([]byte(checksumText))
	}))
	defer server.Close()

	app := &App{store: newTestStore(t)}
	stage, err := app.downloadAndStage(t.Context(), "9.9.9",
		githubAsset{BrowserDownloadURL: server.URL + "/update.zip"},
		githubAsset{BrowserDownloadURL: server.URL + "/update.zip.sha256"}, "", false)
	if err != nil {
		t.Fatalf("downloadAndStage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stage, "Contents", "MacOS", "Verk")); err != nil {
		t.Fatalf("staged application missing: %v", err)
	}
}
