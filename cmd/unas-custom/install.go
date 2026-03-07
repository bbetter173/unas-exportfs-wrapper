package main

import (
	"fmt"
	"os"

	"github.com/bbettridge/unas-custom/internal/installer"
)

func runInstallImpl(args []string) int {
	inst := installer.NewInstaller()
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: install: cannot determine binary path: %v\n", err)
		return 1
	}
	if err := inst.Install(binaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: install: %v\n", err)
		return 1
	}
	fmt.Println("unas-custom: installation complete")
	return 0
}

func runUninstallImpl(args []string) int {
	inst := installer.NewInstaller()
	if err := inst.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: uninstall: %v\n", err)
		return 1
	}
	fmt.Println("unas-custom: uninstall complete")
	return 0
}
