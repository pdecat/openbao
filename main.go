// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main // import "github.com/hashicorp/vault"

import (
	"os"

	"github.com/openbao/openbao/command"
	"github.com/openbao/openbao/internal"
)

func init() {
	// this is a good place to patch SHA-1 support back into x509
	internal.PatchSha1()
}

func main() {
	os.Exit(command.Run(os.Args[1:]))
}
