package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/bbettridge/unas-custom/internal/config"
	"github.com/bbettridge/unas-custom/internal/nfs"
)

const (
	originalExportfs = "/usr/sbin/exportfs.orig"
	configPath       = "/persistent/nfs-intercept/config.yaml"
	exportsDir       = "/etc/exports.d"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := modifyExports(exportsDir, cfg); err != nil {
		return fmt.Errorf("failed to modify exports: %w", err)
	}

	return execOriginal(os.Args[1:])
}

func modifyExports(dir string, cfg *config.Config) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".exports" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		modified, err := nfs.ModifyContent(content, cfg)
		if err != nil {
			return fmt.Errorf("failed to modify %s: %w", path, err)
		}

		if err := os.WriteFile(path, modified, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}

		return nil
	})
}

func execOriginal(args []string) error {
	cmd := exec.Command(originalExportfs, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}

func init() {
	log.SetFlags(0)
	log.SetPrefix("unas-custom: ")

	if _, err := os.Stat(originalExportfs); os.IsNotExist(err) {
		syscall.Exec(originalExportfs, os.Args, os.Environ())
	}
}
