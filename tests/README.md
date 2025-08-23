# Snapshot Testing

## Structure

- `test_dir/` - Contains mock files with various content types for testing
- `snapshots/` - Contains JSON snapshots of expected analysis results
- `snapshot_test.go` - The main snapshot test suite

## Running Tests

### Run snapshot tests (comparison mode)
```bash
make test-snapshots
```

### Update snapshots (baseline mode)
```bash
make test-snapshots-update
```

Or set the environment variable:
```bash
UPDATE_SNAPSHOTS=1 go test -v
```

## Test Cases

The snapshot tests cover various scenarios:

1. **basic_analysis** - Default analysis without dotfiles
2. **include_dotfiles** - Analysis including hidden files
3. **json_output** - JSON format output with metadata
4. **unicode_enabled** - Analysis with Unicode characters enabled
5. **concurrent_processing** - Multi-worker processing

## Test Files

The `test_dir/` contains:
- `main.go` - Go source code with imports and functions
- `config.json` - JSON configuration file
- `README.md` - Markdown documentation
- `data.csv` - CSV data file
- `src/utils.go` - Go utility functions
- `.gitignore` - Gitignore rules
- `ignored.log` - File that should be ignored by gitignore
- `image.svg` - SVG file that should be ignored by extension filtering

## How It Works

1. **Baseline Mode**: When `UPDATE_SNAPSHOTS=1` is set, tests run the analysis and save the results as JSON snapshots
2. **Comparison Mode**: Tests run the analysis and compare results against saved snapshots
3. **Deterministic Sorting**: Character counts are sorted first by count (descending), then by character (ascending) to ensure consistent results
4. **Timing Normalization**: Timing data is zeroed out since it's not deterministic across runs

## When to Update Snapshots

Update snapshots when:
- You intentionally change the analysis algorithm
- You modify the output format
- You add new ignored file types or extensions
- You change the sorting logic

## Debugging Failed Tests

When a snapshot test fails, the output shows both the actual and expected results. Look for differences in:
- Character counts and percentages
- Files found/ignored counts
- Total character counts
- Character distribution changes
