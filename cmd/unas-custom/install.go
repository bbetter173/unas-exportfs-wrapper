package main

import (
	"fmt"
	"os"

	"github.com/bbettridge/unas-custom/internal/installer"
)

var getExecutable = os.Executable
var newInstallerFn = installer.NewInstaller

func runInstallImpl(args []string) int {
	inst := newInstallerFn()
	binaryPath, err := getExecutable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: install: cannot determine binary path: %v\n", err)
		return 1
	}
	return runInstallImplWithInstaller(inst, binaryPath)
}

func runInstallImplWithInstaller(inst *installer.Installer, binaryPath string) int {
	if err := inst.Install(binaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: install: %v\n", err)
		return 1
	}
	fmt.Println("unas-custom: installation complete")
	return 0
}

func runUninstallImpl(args []string) int {
	return runUninstallImplWithInstaller(newInstallerFn())
}

func runUninstallImplWithInstaller(inst *installer.Installer) int {
	if err := inst.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: uninstall: %v\n", err)
		return 1
	}
	fmt.Println("unas-custom: uninstall complete")
	return 0
}
