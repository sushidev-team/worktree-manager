package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const releaseAPI = "https://api.github.com/repos/sushidev-team/worktree-manager/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade wt to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "Checking for updates...")

		// Get latest version
		resp, err := http.Get(releaseAPI)
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}
		defer resp.Body.Close()

		var release githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return fmt.Errorf("failed to parse release info: %w", err)
		}

		latestVersion := strings.TrimPrefix(release.TagName, "v")
		if latestVersion == version {
			fmt.Fprintf(os.Stderr, "Already up to date (v%s)\n", version)
			return nil
		}

		fmt.Fprintf(os.Stderr, "Upgrading v%s → v%s\n", version, latestVersion)

		// Download the new binary
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		filename := fmt.Sprintf("worktree-manager_%s_%s_%s.tar.gz", latestVersion, goos, goarch)
		url := fmt.Sprintf("https://github.com/sushidev-team/worktree-manager/releases/download/v%s/%s", latestVersion, filename)

		resp, err = http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: HTTP %d (no release for %s/%s?)", resp.StatusCode, goos, goarch)
		}

		// Extract the binary from the tar.gz
		binary, err := extractBinaryFromTarGz(resp.Body, "wt")
		if err != nil {
			return fmt.Errorf("failed to extract binary: %w", err)
		}

		// Replace the current executable
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find current executable: %w", err)
		}

		// Write to a temp file next to the current binary, then rename (atomic)
		tmpPath := execPath + ".new"
		if err := os.WriteFile(tmpPath, binary, 0o755); err != nil {
			return fmt.Errorf("failed to write new binary: %w", err)
		}

		if err := os.Rename(tmpPath, execPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to replace binary (try with sudo): %w", err)
		}

		fmt.Fprintf(os.Stderr, "Upgraded to v%s\n", latestVersion)
		return nil
	},
}

func extractBinaryFromTarGz(r io.Reader, name string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Name == name {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %q not found in archive", name)
}
