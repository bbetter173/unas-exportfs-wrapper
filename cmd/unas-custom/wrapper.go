package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bbettridge/unas-custom/internal/config"
	"github.com/bbettridge/unas-custom/internal/logging"
	"github.com/bbettridge/unas-custom/internal/nfs"
	"github.com/bbettridge/unas-custom/internal/smb"
)

const (
	nfsOriginalPath = "/usr/sbin/exportfs.orig"
	nfsConfigPath   = "/persistent/unas-custom/config.yaml"
	nfsExportsDir   = "/etc/exports.d"
	smbOriginalPath = "/usr/bin/smbcontrol.orig"

	wrapperLogPath = "/persistent/unas-custom/unas-custom.log"
)

func runNfsWrapper(args []string) int {
	return runNfsWrapperWithPaths(args, nfsConfigPath, nfsExportsDir, nfsOriginalPath)
}

func runNfsWrapperWithPaths(args []string, configPath, exportsDir, originalPath string) int {
	logger := newWrapperLogger()
	defer logger.Close()

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Warn("nfs wrapper: config load failed from %s (fail-open): %v", configPath, err)
		return runOriginalCommand(originalPath, args)
	}

	if err := rewriteExportsFiles(exportsDir, cfg); err != nil {
		logger.Error("nfs wrapper: failed to modify exports in %s (fail-open): %v", exportsDir, err)
		return runOriginalCommand(originalPath, args)
	}

	return runOriginalCommand(originalPath, args)
}

func runSmbcontrolWrapper(args []string) int {
	return runSmbcontrolWrapperWithPaths(args, smb.DefaultSmbConfPath, smb.DefaultIncludeLine, smbOriginalPath)
}

func runSmbcontrolWrapperWithPaths(args []string, smbConfPath, includeLine, originalPath string) int {
	logger := newWrapperLogger()
	defer logger.Close()

	if err := smb.Inject(smbConfPath, includeLine); err != nil {
		logger.Warn("smb wrapper: include injection failed for %s (fail-open): %v", smbConfPath, err)
	}

	return runOriginalCommand(originalPath, args)
}

func rewriteExportsFiles(exportsDir string, cfg *config.Config) error {
	type exportUpdate struct {
		path    string
		content []byte
	}

	updates := make([]exportUpdate, 0)
	err := filepath.WalkDir(exportsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".exports") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		updated, err := nfs.ModifyContent(content, cfg)
		if err != nil {
			return fmt.Errorf("modify %s: %w", path, err)
		}

		updates = append(updates, exportUpdate{path: path, content: updated})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, update := range updates {
		if err := os.WriteFile(update.path, update.content, 0644); err != nil {
			return fmt.Errorf("write %s: %w", update.path, err)
		}
	}

	return nil
}

func runOriginalCommand(originalPath string, args []string) int {
	if _, err := os.Stat(originalPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "unas-custom: %s not found — reinstall with unas-custom install\n", originalPath)
			return 1
		}
		fmt.Fprintf(os.Stderr, "unas-custom: cannot access %s: %v\n", originalPath, err)
		return 1
	}

	cmd := exec.Command(originalPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}

	return 0
}

func newWrapperLogger() *logging.Logger {
	logger, err := logging.New(wrapperLogPath)
	if err != nil {
		return logging.NewNoop()
	}
	return logger
}
