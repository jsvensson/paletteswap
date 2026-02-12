package main

import (
	"fmt"
	"os"

	"github.com/jsvensson/paletteswap"
	"github.com/jsvensson/paletteswap/internal/format"
	"github.com/spf13/cobra"
)

var (
	flagTheme     string
	flagOut       string
	flagTemplates string
	flagApp       []string
	flagCheck     bool
	version       = "dev" // Injected at build time via ldflags
)

var rootCmd = &cobra.Command{
	Use:     "paletteswap",
	Short:   "Generate application-specific color themes from a single HCL source file",
	Version: version,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate theme files from templates",
	RunE:  runGenerate,
}

var fmtCmd = &cobra.Command{
	Use:   "fmt [files...]",
	Short: "Format .pstheme files",
	Long:  "Format one or more .pstheme files in-place. Prints the name of each file that was modified.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runFmt,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), version)
	},
}

func init() {
	generateCmd.Flags().StringVar(&flagTheme, "theme", "theme.hcl", "path to theme HCL file")
	generateCmd.Flags().StringVar(&flagOut, "out", "output", "output directory")
	generateCmd.Flags().StringVar(&flagTemplates, "templates", "templates", "templates directory")
	generateCmd.Flags().StringArrayVar(&flagApp, "app", nil, "generate only for specific apps (can be repeated)")
	fmtCmd.Flags().BoolVarP(&flagCheck, "check", "c", false, "check if files are formatted (do not write changes)")
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(fmtCmd)
	rootCmd.AddCommand(versionCmd)
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

func runFmt(cmd *cobra.Command, args []string) error {
	hasErrors := false
	needsFormatting := false

	for _, path := range args {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error reading %s: %v\n", path, err)
			hasErrors = true
			continue
		}

		content := string(data)
		formatted, err := format.Format(content)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error formatting %s: %v\n", path, err)
			hasErrors = true
			continue
		}

		if formatted == content {
			continue
		}

		fmt.Fprintln(cmd.OutOrStdout(), path)
		needsFormatting = true

		if !flagCheck {
			if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error writing %s: %v\n", path, err)
				hasErrors = true
			}
		}
	}

	if hasErrors || (flagCheck && needsFormatting) {
		os.Exit(1)
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
