# Character Sequence Counting Feature

## Overview
Add character sequence analysis to symbolista, counting 2-N character sequences that developers frequently type together, providing insights into coding patterns and common character combinations.

## Core Requirements

### Sequence Definition
- **Length**: Configurable 2-N characters (default 2-3)
- **Normalization**: All lowercase 
- **Whitespace Handling**: Skip whitespace characters entirely during sequence extraction
- **Occurrence Threshold**: Only count sequences appearing ≥ threshold times (default: 2)

### Example Extraction
```javascript
// Input: "const foo: { \n  bar"
// Whitespace positions: [10] (space), [12] (space), [13] (newline), [14-15] (spaces)
// Character stream (ignoring whitespace): "constfoo:{bar"
// 2-char sequences: "co", "on", "ns", "st", "tf", "fo", "oo", "o:", ":{", "{b", "ba", "ar"
// 3-char sequences: "con", "ons", "nst", "stf", "tfo", "foo", "oo:", "o:{", ":{b", "{ba", "bar"
```

## Algorithm Design

### Core Parsing Algorithm
```go
func extractSequences(content string, minLength, maxLength int, threshold int) map[string]int {
    // Convert to lowercase runes, filter out whitespace in single pass
    var cleanRunes []rune
    for _, r := range strings.ToLower(content) {
        if !unicode.IsSpace(r) {
            cleanRunes = append(cleanRunes, r)
        }
    }
    
    sequenceMap := make(map[string]int)
    
    // Sliding window extraction
    for length := minLength; length <= maxLength; length++ {
        for i := 0; i <= len(cleanRunes)-length; i++ {
            seq := string(cleanRunes[i:i+length])
            if isValidSequence(seq, asciiOnly) {
                sequenceMap[seq]++
            }
        }
    }
    
    // Apply occurrence threshold
    for seq, count := range sequenceMap {
        if count < threshold {
            delete(sequenceMap, seq)
        }
    }
    
    return sequenceMap
}
```

### Validation Rules
```go
func isValidSequence(seq string, asciiOnly bool) bool {
    if len(seq) < 2 {
        return false
    }
    
    for _, r := range seq {
        // Must be graphic or control character (no spaces since we filtered)
        if !unicode.IsGraphic(r) && !unicode.IsControl(r) {
            return false
        }
        
        // ASCII-only filtering
        if asciiOnly && r > 127 {
            return false
        }
    }
    
    return true
}
```

## Edge Cases & Accuracy Considerations

### 1. **Cross-Boundary Artifacts**
- **Issue**: File boundaries create artificial sequence breaks
- **Solution**: Accept limitation (consistent with character counting)
- **Impact**: Minimal - most meaningful sequences occur within files

### 2. **Whitespace Collapsing Effects**
- **Issue**: `"a   b"` becomes `"ab"` - different from what user typed
- **Rationale**: We want typing patterns, not whitespace patterns
- **Validation**: `": {"` → `":{"`represents the typing pattern correctly

### 3. **String Literal Content**
- **Issue**: Sequences inside strings may not represent typing patterns
- **Solution**: Accept - filtering would require language parsing
- **Mitigation**: Occurrence threshold filters noise

### 4. **Repetitive Code Patterns**
- **Issue**: Auto-generated or template code creates sequence bias
- **Example**: 1000 occurrences of `"qu"` in SQL queries vs natural frequency
- **Solution**: Consider logarithmic weighting or capping per-file contributions
- **Current**: Accept raw counts for simplicity

### 5. **Unicode Normalization**
- **Issue**: `"café"` vs `"cafe\u0301"` (composed vs decomposed)
- **Solution**: No normalization beyond lowercase - rare in source code
- **Assumption**: Source code primarily uses composed Unicode

### 6. **Memory Explosion Scenarios**
- **Issue**: Large files with many unique sequences
- **Mitigation**: Occurrence threshold + per-file limits
- **Limits**: 
  - Max 50,000 unique sequences per file
  - Early termination if exceeded

### 7. **ASCII vs Unicode Consistency**
- **Issue**: ASCII mode should be consistent between chars and sequences
- **Solution**: Apply ASCII filtering at rune level during validation
- **Behavior**: Unicode sequences filtered out in ASCII mode

## Performance Optimizations

### Memory Management
- Pre-filter whitespace to reduce string operations
- Use `strings.Builder` for sequence construction
- Apply threshold filtering during collection, not after

### Worker Integration
- Each worker maintains local sequence map
- Result collector merges maps using efficient algorithms
- No cross-worker synchronization needed

## Configuration Options

### CLI Flags
- `--sequence-min-length=2`: Minimum sequence length
- `--sequence-max-length=3`: Maximum sequence length  
- `--sequence-threshold=2`: Minimum occurrence count
- `--enable-sequences`: Enable sequence counting (default: false)

### TUI Controls
- `v`: Toggle sequence view mode
- Same filtering controls as characters (a/l/s/w)

## Output Format Changes

### JSON Structure
```json
{
  "result": {
    "characters": [...],
    "sequences": [
      {
        "sequence": ":{",
        "count": 1247,
        "percentage": 0.84
      }
    ]
  }
}
```

### Table Output
```
Characters:
-----------
Character  Count      Percentage
{...}

Sequences (2-3 chars):
---------------------
Sequence   Count      Percentage
:{         1247       0.84%
```

### CSV Output
```csv
type,sequence,count,percentage
character,a,12543,8.9%
sequence,:{,1247,0.84%
```

## Implementation Priority

### Phase 1: Core Functionality
1. Extend worker result structures for sequences
2. Implement sequence extraction algorithm
3. Update result collector for sequence merging
4. Add sequence counting to analysis pipeline

### Phase 2: Output Integration
1. Update JSON output format
2. Extend table output format
3. Modify CSV output format

### Phase 3: TUI Integration
1. Add sequence display mode
2. Implement 'v' toggle hotkey
3. Apply existing filters to sequences
4. Add sequence-specific stats display

### Phase 4: Configuration
1. Add CLI flags for sequence parameters
2. Validate parameter combinations
3. Update help documentation

## Testing Strategy

### Unit Tests
- Whitespace filtering accuracy
- Sequence extraction correctness
- Threshold application
- Unicode/ASCII mode consistency

### Integration Tests
- End-to-end sequence counting
- Output format validation
- TUI sequence mode functionality
- Performance with large codebases

### Edge Case Tests
- Empty files
- Files with only whitespace
- Very long lines
- Mixed Unicode content
- Memory limit scenarios