package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ogdakke/symbolista/internal/counter"
	"github.com/ogdakke/symbolista/internal/logger"
	"github.com/ogdakke/symbolista/internal/tui"
	"github.com/spf13/cobra"
)

const Version = "v0.0.10"

var (
	outputFormat    string
	showPercentages bool
	verboseCount    int
	workerCount     int
	includeDotfiles bool
	asciiOnly       bool
	useTUI          bool
	showVersion     bool
	includeMetadata bool
)

var rootCmd = &cobra.Command{
	Use:   "symbolista [directory]",
	Short: "Count symbols and characters in a codebase",
	Long: `Symbolista recursively counts symbols and characters in a codebase,
respecting gitignore rules and outputting the most used characters with counts and percentages.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Println(Version)
			return
		}

		if len(args) == 0 {
			cmd.Help()
			return
		}

		startTime := time.Now()
		logger.SetVerbosity(verboseCount)

		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		if useTUI {
			logger.Info("Starting TUI mode", "directory", dir, "verbosity", verboseCount, "workers", workerCount, "includeDotfiles", includeDotfiles, "asciiOnly", asciiOnly)
			err := tui.RunTUI(dir, showPercentages, workerCount, includeDotfiles, asciiOnly)
			if err != nil {
				fmt.Printf("TUI error: %v\n", err)
				os.Exit(1)
			}
			return
		}

		logger.Info("Starting symbol analysis", "directory", dir, "format", outputFormat, "verbosity", verboseCount, "workers", workerCount, "includeDotfiles", includeDotfiles, "asciiOnly", asciiOnly)
		counter.CountSymbolsConcurrent(dir, outputFormat, showPercentages, workerCount, includeDotfiles, asciiOnly, includeMetadata)

		totalExecutionTime := time.Since(startTime)
		if verboseCount > 0 {
			logger.Info("Total execution time", "duration", totalExecutionTime)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version and exit")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table, json, csv)")
	rootCmd.Flags().BoolVarP(&showPercentages, "percentages", "p", true, "Show percentages in output")
	rootCmd.Flags().CountVarP(&verboseCount, "verbose", "V", "Increase verbosity (-V info, -VV debug, -VVV trace)")
	rootCmd.Flags().IntVarP(&workerCount, "workers", "w", 0, "Number of worker goroutines (0 = auto-detect based on CPU cores) (default 0)")
	rootCmd.Flags().BoolVar(&includeDotfiles, "include-dotfiles", false, "Include dotfiles in analysis (default false)")
	rootCmd.Flags().BoolVar(&asciiOnly, "ascii-only", true, "Count only ASCII characters. Use --ascii-only=false to include all Unicode characters")
	rootCmd.Flags().BoolVar(&useTUI, "tui", false, "Launch interactive TUI interface")
	rootCmd.Flags().BoolVarP(&includeMetadata, "metadata", "m", true, "Include metadata in JSON output (directory, file counts, timing info) (default true)")
}
