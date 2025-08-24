package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ogdakke/symbolista/internal/domain"
)

type Outputter struct {
}

func NewOutputter() *Outputter {
	return &Outputter{}
}

func (o *Outputter) Output(
	kind string,
	result domain.AnalysisResult,
	showPercentages bool,
	directory string,
	includeMetadata bool,
) {
	switch kind {
	case "json":

		o.OutputJSON(showPercentages, directory, result, includeMetadata)
	case "csv":

		o.OutputCSV(result.CharCounts, result.SequenceCounts, showPercentages)
	default:

		o.OutputTable(result.CharCounts, result.SequenceCounts, showPercentages)
	}
}

func (o *Outputter) OutputTable(
	counts domain.CharCounts,
	sequences domain.SequenceCounts,
	showPercentages bool,
) {
	width := 35
	fmt.Println("Characters:")
	fmt.Println(strings.Repeat("-", width))
	fmt.Printf("%-10s %-10s", "Character", "Count")
	if showPercentages {
		fmt.Printf(" %-12s", "Percentage")
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", width))

	formatChars(counts, func(char string, count int, percentage float64) {
		fmt.Printf("%-10s %-10d", char, count)
		if showPercentages {
			fmt.Printf(" %-12.2f%%", percentage)
		}
		fmt.Println()
	})
	fmt.Println(strings.Repeat("-", width))

	if len(sequences) > 0 {
		seqs := formatSequences(sequences)
		fmt.Printf("\nSequences (2-3 chars):\n")
		fmt.Println(strings.Repeat("-", width))
		fmt.Printf("%-10s %-10s", "Sequence", "Count")
		if showPercentages {
			fmt.Printf(" %-12s", "Percentage")
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", width))

		for _, seq := range seqs {
			fmt.Printf("%-10s %-10d", seq.Sequence, seq.Count)
			if showPercentages {
				fmt.Printf(" %-12.2f%%", seq.Percentage)
			}
			fmt.Println()
		}
		fmt.Println(strings.Repeat("-", width))
	}
}

func (o *Outputter) OutputCSV(
	counts domain.CharCounts,
	sequences domain.SequenceCounts,
	showPercentages bool,
) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := []string{"type", "sequence", "count"}
	if showPercentages {
		headers = append(headers, "percentage")
	}
	writer.Write(headers)

	formatChars(counts, func(char string, count int, percentage float64) {
		row := []string{"character", char, fmt.Sprintf("%d", count)}
		if showPercentages {
			row = append(row, fmt.Sprintf("%.2f%%", percentage))
		}
		writer.Write(row)
	})

	seqs := formatSequences(sequences)
	for _, seq := range seqs {
		row := []string{"sequence", seq.Sequence, fmt.Sprintf("%d", seq.Count)}
		if showPercentages {
			row = append(row, fmt.Sprintf("%.2f%%", seq.Percentage))
		}
		writer.Write(row)
	}
}

func (o *Outputter) OutputJSON(
	showPercentages bool,
	directory string,
	result domain.AnalysisResult,
	includeMetadata bool,
) {
	counts := result.CharCounts

	if !showPercentages {
		for i := range counts {
			counts[i].Percentage = 0
		}
	}

	output := domain.JSONOutput{
		Result: domain.JSONResult{
			Characters: counts,
			Sequences:  result.SequenceCounts,
		},
	}

	if includeMetadata {
		output.Metadata = &domain.JSONMetadata{
			Directory:       directory,
			FilesFound:      result.FilesFound,
			FilesProcessed:  result.FilesFound - result.FilesIgnored,
			FilesIgnored:    result.FilesIgnored,
			TotalCharacters: result.TotalChars,
			UniqueChars:     result.UniqueChars,
			Timing:          result.Timing,
		}
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

type OnCharFunc func(char string, count int, percentage float64)

func formatChars(counts domain.CharCounts, onChar OnCharFunc) domain.CharCounts {
	for _, c := range counts {
		char := c.Char
		switch c.Char {
		case " ":
			char = "<space>"
		case "\t":
			char = "<tab>"
		case "\n":
			char = "<newline>"
		case "\r":
			char = "<return>"
		case "\f":
			char = "<formfeed>"
		case "\v":
			char = "<vert_tab>"
		}

		onChar(char, c.Count, c.Percentage)
	}
	return counts
}

func formatSequences(seqs domain.SequenceCounts) domain.SequenceCounts {
	var sequencesFormatted domain.SequenceCounts = make(domain.SequenceCounts, 0)
	for _, seq := range seqs {
		seq.Sequence = strings.ReplaceAll(seq.Sequence, "\n", "↵")
		seq.Sequence = strings.ReplaceAll(seq.Sequence, " ", "⎵")
		seq.Sequence = strings.ReplaceAll(seq.Sequence, "\t", "⇥")
		seq.Sequence = strings.ReplaceAll(seq.Sequence, "\r", "⏎")
		sequencesFormatted = append(sequencesFormatted, seq)
	}
	return sequencesFormatted
}
