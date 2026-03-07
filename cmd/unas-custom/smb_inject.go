package main

import (
	"fmt"
	"os"

	"github.com/bbettridge/unas-custom/internal/smb"
)

// runSmbInjectImpl implements the "smb inject" subcommand.
// Idempotently injects the include line into smb.conf.
// Returns 0 on success, 1 on error.
func runSmbInjectImpl(args []string) int {
	return runSmbInjectImplWithPath(args, smb.DefaultSmbConfPath)
}

func runSmbInjectImplWithPath(args []string, smbConfPath string) int {
	if err := smb.Inject(smbConfPath, smb.DefaultIncludeLine); err != nil {
		fmt.Fprintf(os.Stderr, "unas-custom: smb inject: %v\n", err)
		return 1
	}
	fmt.Println("smb inject: include line present in smb.conf")
	return 0
}
