package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// version is the bitbucket-cli release version. Overridable at build time via
// -ldflags "-X bitbucket-cli/cmd.version=...".
var version = "0.1.0"

// Persistent flags shared by all subcommands.
var (
	flagWorkspace string
	flagRepo      string
	flagPretty    bool
)

var rootCmd = &cobra.Command{
	Use:           "bitbucket-cli",
	Short:         "Bitbucket Cloud CLI for agents and humans",
	Long:          "bitbucket-cli exposes Bitbucket Cloud pull requests, branches, and repo\ninfo as scriptable commands. Output is JSON by default; pass --pretty for\nhuman-readable text. Credentials come from environment variables.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagWorkspace, "workspace", "", "Bitbucket workspace slug (defaults to BITBUCKET_DEFAULT_WORKSPACE or git remote)")
	pf.StringVar(&flagRepo, "repo", "", "Bitbucket repository slug (defaults to BITBUCKET_DEFAULT_REPO or git remote)")
	pf.BoolVar(&flagPretty, "pretty", false, "Render human-readable text instead of JSON")
}
