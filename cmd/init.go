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
yaml-language-server schema header, so editors validate and complete it.

--from copies one of the tested example campaigns instead: its fleet.yaml,
scripts and test cases, into the directory of --file. A name and --org, when
given, replace the example's name and target org. --list shows the examples.`,
	Example: `  gh fleet init pin-actions
  gh fleet init bump-go --org acme --file campaigns/bump-go.yaml
  gh fleet init --list
  gh fleet init node-22 --from node-version --org acme --file campaigns/node-22/fleet.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		o := app.InitOptions{
			File:  viper.GetString("file"),
			Org:   viper.GetString("org"),
			Force: viper.GetBool("force"),
			From:  viper.GetString("from"),
			List:  viper.GetBool("list"),
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
	initCmd.Flags().Bool("force", false, "overwrite existing files")
	initCmd.Flags().String("from", "", "copy this example campaign, with its scripts and test cases")
	initCmd.Flags().Bool("list", false, "list the example campaigns --from can copy")
	rootCmd.AddCommand(initCmd)
}
