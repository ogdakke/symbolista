package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ogdakke/symbolista/internal/counter"
	"github.com/ogdakke/symbolista/internal/logger"
	"github.com/spf13/cobra"
)

var (
	directory       string
	outputFormat    string
	showPercentages bool
	verboseCount    int
)

var rootCmd = &cobra.Command{
	Use:   "symbolista [directory]",
	Short: "Count symbols and characters in a codebase",
	Long: `Symbolista recursively counts symbols and characters in a codebase,
respecting gitignore rules and outputting the most used characters with counts and percentages.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		logger.SetVerbosity(verboseCount)

		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		if directory != "" {
			dir = directory
		}

		logger.Info("Starting symbol analysis", "directory", dir, "format", outputFormat, "verbosity", verboseCount)
		counter.CountSymbols(dir, outputFormat, showPercentages)

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
	rootCmd.Flags().StringVarP(&directory, "directory", "d", "", "Directory to analyze")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table, json, csv)")
	rootCmd.Flags().BoolVarP(&showPercentages, "percentages", "p", true, "Show percentages in output")
	rootCmd.Flags().CountVarP(&verboseCount, "verbose", "V", "Increase verbosity (-V info, -VV debug, -VVV trace)")
}
