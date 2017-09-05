package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
)

// Ensure we are implementing the right interfaces.
var _ cli.Command = (*MountCommand)(nil)
var _ cli.CommandAutocomplete = (*MountCommand)(nil)

// MountCommand is a Command that mounts a new mount.
type MountCommand struct {
	*BaseCommand

	flagDescription     string
	flagPath            string
	flagDefaultLeaseTTL time.Duration
	flagMaxLeaseTTL     time.Duration
	flagForceNoCache    bool
	flagPluginName      string
	flagLocal           bool
}

func (c *MountCommand) Synopsis() string {
	return "Mounts a secret backend at a path"
}

func (c *MountCommand) Help() string {
	helpText := `
Usage: vault mount [options] TYPE

  Mount a secret backend at a particular path. By default, secret backends are
  mounted at the path corresponding to their "type", but users can customize
  the mount point using the -path option.

  Once mounted at a path, Vault will route all requests which begin with the
  path to the secret backend.

  Mount the AWS backend at aws/:

      $ vault mount aws

  Mount the SSH backend at ssh-prod/:

      $ vault mount -path=ssh-prod ssh

  Mount the database backend with an explicit maximum TTL of 30m:

      $ vault mount -max-lease-ttl=30m database

  Mount a custom plugin (after it is registered in the plugin registry):

      $ vault mount -path=my-secrets -plugin-name=my-custom-plugin plugin

  For a full list of secret backends and examples, please see the documentation.

` + c.Flags().Help()

	return strings.TrimSpace(helpText)
}

func (c *MountCommand) Flags() *FlagSets {
	set := c.flagSet(FlagSetHTTP)

	f := set.NewFlagSet("Command Options")

	f.StringVar(&StringVar{
		Name:       "description",
		Target:     &c.flagDescription,
		Completion: complete.PredictAnything,
		Usage:      "Human-friendly description for the purpose of this mount.",
	})

	f.StringVar(&StringVar{
		Name:       "path",
		Target:     &c.flagPath,
		Default:    "", // The default is complex, so we have to manually document
		Completion: complete.PredictAnything,
		Usage: "Place where the mount will be accessible. This must be " +
			"unique across all mounts. This defaults to the \"type\" of the mount.",
	})

	f.DurationVar(&DurationVar{
		Name:       "default-lease-ttl",
		Target:     &c.flagDefaultLeaseTTL,
		Completion: complete.PredictAnything,
		Usage: "The default lease TTL for this backend. If unspecified, this " +
			"defaults to the Vault server's globally configured default lease TTL.",
	})

	f.DurationVar(&DurationVar{
		Name:       "max-lease-ttl",
		Target:     &c.flagMaxLeaseTTL,
		Completion: complete.PredictAnything,
		Usage: "The maximum lease TTL for this backend. If unspecified, this " +
			"defaults to the Vault server's globally configured maximum lease TTL.",
	})

	f.BoolVar(&BoolVar{
		Name:    "force-no-cache",
		Target:  &c.flagForceNoCache,
		Default: false,
		Usage: "Force the backend to disable caching. If unspecified, this " +
			"defaults to the Vault server's globally configured cache settings. " +
			"This does not affect caching of the underlying encrypted data storage.",
	})

	f.StringVar(&StringVar{
		Name:       "plugin-name",
		Target:     &c.flagPluginName,
		Completion: complete.PredictAnything,
		Usage: "Name of the plugin to mount. This plugin name must already exist " +
			"in the Vault server's plugin catalog.",
	})

	f.BoolVar(&BoolVar{
		Name:    "local",
		Target:  &c.flagLocal,
		Default: false,
		Usage: "Mark the mount as a local-only mount. Local mounts are not " +
			"replicated nor removed by replication.",
	})

	return set
}

func (c *MountCommand) AutocompleteArgs() complete.Predictor {
	return c.PredictVaultAvailableMounts()
}

func (c *MountCommand) AutocompleteFlags() complete.Flags {
	return c.Flags().Completions()
}

func (c *MountCommand) Run(args []string) int {
	f := c.Flags()

	if err := f.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	args = f.Args()
	switch len(args) {
	case 0:
		c.UI.Error("Missing TYPE!")
		return 1
	case 1:
		// OK
	default:
		c.UI.Error(fmt.Sprintf("Too many arguments (expected 1, got %d)", len(args)))
		return 1
	}

	client, err := c.Client()
	if err != nil {
		c.UI.Error(err.Error())
		return 2
	}

	// Get the mount type (first arg)
	mountType := strings.TrimSpace(args[0])

	// If no path is specified, we default the path to the backend type
	// or use the plugin name if it's a plugin backend
	mountPath := c.flagPath
	if mountPath == "" {
		if mountType == "plugin" {
			mountPath = c.flagPluginName
		} else {
			mountPath = mountType
		}
	}

	// Append a trailing slash to indicate it's a path in output
	mountPath = ensureTrailingSlash(mountPath)

	// Build mount input
	mountInput := &api.MountInput{
		Type:        mountType,
		Description: c.flagDescription,
		Local:       c.flagLocal,
		Config: api.MountConfigInput{
			DefaultLeaseTTL: c.flagDefaultLeaseTTL.String(),
			MaxLeaseTTL:     c.flagMaxLeaseTTL.String(),
			ForceNoCache:    c.flagForceNoCache,
			PluginName:      c.flagPluginName,
		},
	}

	if err := client.Sys().Mount(mountPath, mountInput); err != nil {
		c.UI.Error(fmt.Sprintf("Error mounting: %s", err))
		return 2
	}

	mountThing := mountType + " secret backend"
	if mountType == "plugin" {
		mountThing = c.flagPluginName + " plugin"
	}

	c.UI.Output(fmt.Sprintf("Success! Mounted the %s at: %s", mountThing, mountPath))
	return 0
}
