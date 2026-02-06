package main

import (
	"fmt"
	"os"

	"github.com/jsvensson/paletteswap"
	"github.com/spf13/cobra"
)

var (
	flagTheme     string
	flagOut       string
	flagTemplates string
	flagApp       []string
)

var rootCmd = &cobra.Command{
	Use:   "paletteswap",
	Short: "Generate application-specific color themes from a single HCL source file",
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate theme files from templates",
	RunE:  runGenerate,
}

func init() {
	generateCmd.Flags().StringVar(&flagTheme, "theme", "theme.hcl", "path to theme HCL file")
	generateCmd.Flags().StringVar(&flagOut, "out", "output", "output directory")
	generateCmd.Flags().StringVar(&flagTemplates, "templates", "templates", "templates directory")
	generateCmd.Flags().StringArrayVar(&flagApp, "app", nil, "generate only for specific apps (can be repeated)")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	theme, err := paletteswap.Load(flagTheme)
	if err != nil {
		return fmt.Errorf("loading theme: %w", err)
	}

	e := &paletteswap.Engine{
		TemplatesDir: flagTemplates,
		OutputDir:    flagOut,
		Apps:         flagApp,
	}

	if err := e.Run(theme); err != nil {
		return fmt.Errorf("generating: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Generated theme files in %s\n", flagOut)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
