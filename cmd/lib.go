package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/star"
)

// libCmd prints the built-in Starlark library's docs.
var libCmd = &cobra.Command{
	Use:   "lib [module[.function]] [function]",
	Short: "Browse the built-in Starlark library: modules, functions and examples",
	Long: `Print the docs of the @fleet// Starlark library that ships in this binary,
so they always match the version you run.

With no arguments, lists the modules. With a module, such as docker, prints
what it does, its load line and its functions. With a function, such as
docker.bump_image, prints its arguments, what it returns and its example,
which the library's tests run, so it shows real output.

--search finds functions whose names or docs mention every word given.
--source prints a module's Starlark source.`,
	Example: `  gh fleet lib
  gh fleet lib docker
  gh fleet lib docker.bump_image
  gh fleet lib --search "never lower"
  gh fleet lib refs --source`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		o := app.LibOptions{
			Common: app.Common{
				JSON: viper.GetString("json") != "", JSONFields: jsonFields(viper.GetString("json")),
				JQ: viper.GetString("jq"), Template: viper.GetString("template"),
			},
			Args:   args,
			Search: viper.GetString("search"),
			Source: viper.GetBool("source"),
		}
		return run(cmd, func(a *app.App) error { return a.Lib(o) })
	},
}

// init registers the lib command and its flags.
func init() {
	libCmd.Flags().StringP("search", "s", "", "list functions whose names or docs mention these words")
	libCmd.Flags().Bool("source", false, "print the module's Starlark source")
	addOutputFlags(libCmd, []app.LibModule{}, []app.LibHit{}, star.LibDoc{}, star.LibFunc{})
	rootCmd.AddCommand(libCmd)
}
