package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	switch detectMode() {
	case "nfs-wrapper":
		os.Exit(runNfsWrapper(os.Args[1:]))
	case "smb-wrapper":
		os.Exit(runSmbcontrolWrapper(os.Args[1:]))
	default:
		os.Exit(runCLI(os.Args[1:]))
	}
}

func detectMode() string {
	name := filepath.Base(os.Args[0])
	switch name {
	case "exportfs":
		return "nfs-wrapper"
	case "smbcontrol":
		return "smb-wrapper"
	default:
		return "cli"
	}
}

func runCLI(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return 0
	}

	cmd := parseSubcommand(args)
	switch cmd {
	case "install":
		return runInstall(args[1:])
	case "uninstall":
		return runUninstall(args[1:])
	case "status":
		return runStatus(args[1:])
	case "smb-apply":
		return runSmbApply(args[2:])
	case "smb-inject":
		return runSmbInject(args[2:])
	case "help":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unas-custom: unknown command %q\n\n", args[0])
		printHelp()
		return 1
	}
}

func parseSubcommand(args []string) string {
	if len(args) == 0 {
		return "help"
	}

	if args[0] == "--help" || args[0] == "-h" {
		return "help"
	}

	if args[0] == "smb" && len(args) >= 2 {
		switch args[1] {
		case "apply":
			return "smb-apply"
		case "inject":
			return "smb-inject"
		}
	}

	switch args[0] {
	case "install", "uninstall", "status":
		return args[0]
	}

	return "unknown"
}

func printHelp() {
	fmt.Print(`unas-custom: UniFi NAS customization tool

  Usage: unas-custom <command>

  Commands:
    install       Install NFS + SMB hooks
    uninstall     Remove all hooks
    status        Show installation status
    smb apply     Generate SMB override file from config
    smb inject    Inject include line into smb.conf (idempotent)

  Wrapper modes (invoked via symlink):
    exportfs      NFS wrapper mode (intercepts exportfs calls)
    smbcontrol    SMB wrapper mode (intercepts smbcontrol calls)
`)
}

func runInstall(args []string) int { return runInstallImpl(args) }

func runUninstall(args []string) int { return runUninstallImpl(args) }

func runStatus(args []string) int {
	return runStatusImpl(args)
}

func runSmbApply(args []string) int {
	return runSmbApplyWithPaths(args, smbApplyDefaultConfigPath, smbApplyDefaultOverridesPath, smbApplyDefaultShareConfPath)
}

func runSmbInject(args []string) int {
	return runSmbInjectImpl(args)
}
