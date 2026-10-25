package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
)

// initCmd scaffolds a campaign file.
var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold a campaign file with commented defaults",
	Long: `Write a new campaign file with every option shown and commented.

The name defaults to the current directory's name. The file starts with a
yaml-language-server schema header, so editors validate and complete it.`,
	Example: `  gh fleet init pin-actions
  gh fleet init bump-go --org acme --file campaigns/bump-go.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o := app.InitOptions{
			File:  viper.GetString("file"),
			Org:   viper.GetString("org"),
			Force: viper.GetBool("force"),
		}
		if len(args) > 0 {
			o.Name = args[0]
		}
		return run(cmd, func(a *app.App) error { return a.Init(o) })
	},
}

// init registers the init command and its flags.
func init() {
	initCmd.Flags().StringP("file", "f", "", "where to write the campaign (default ./fleet.yaml)")
	initCmd.Flags().String("org", "", "organization to target")
	initCmd.Flags().Bool("force", false, "overwrite an existing file")
	rootCmd.AddCommand(initCmd)
}
