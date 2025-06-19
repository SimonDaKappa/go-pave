#!/bin/bash

# Advanced script to generate static flamegraph SVG files
# Usage: ./generate_flamegraph.sh [benchmark_pattern] [output_name]

set -e

BENCHMARK_PATTERN=${1:-"."}
OUTPUT_NAME=${2:-"benchmark"}
OUTPUT_DIR="profiles"

echo "Creating output directory..."
mkdir -p $OUTPUT_DIR

echo "Running benchmarks with profiling..."
go test -bench=$BENCHMARK_PATTERN -cpuprofile=$OUTPUT_DIR/cpu.prof -memprofile=$OUTPUT_DIR/mem.prof -benchtime=10s

echo "Generating flamegraph data..."
go tool pprof -raw -output=$OUTPUT_DIR/cpu_raw.txt $OUTPUT_DIR/cpu.prof

echo "Processing raw profile data for flamegraph..."
# Create a more compatible format for flamegraph
go tool pprof -text $OUTPUT_DIR/cpu.prof | \
  grep -v "^Showing nodes" | \
  grep -v "^See" | \
  grep -v "^Total" | \
  grep -v "^$" | \
  awk 'NF>=2 {
    # Extract the function name and time
    time = $1
    gsub(/s$/, "", time)  # Remove 's' suffix
    gsub(/%/, "", time)   # Remove '%' if present
    
    # Get function name (everything after the first two fields)
    func = ""
    for(i=3; i<=NF; i++) {
      if(i>3) func = func " "
      func = func $i
    }
    
    # Convert percentage to actual samples (approximate)
    samples = int(time * 100)
    if(samples > 0) {
      for(j=0; j<samples; j++) {
        print func
      }
    }
  }' > $OUTPUT_DIR/flamegraph_data.txt

# Alternative: Use go-torch if available
if command -v go-torch &> /dev/null; then
    echo "Generating flamegraph with go-torch..."
    go-torch -f $OUTPUT_DIR/${OUTPUT_NAME}_flamegraph.svg $OUTPUT_DIR/cpu.prof
    echo "Flamegraph saved as: $OUTPUT_DIR/${OUTPUT_NAME}_flamegraph.svg"
else
    echo "go-torch not found. Using pprof web interface instead."
    echo "Run: go tool pprof -http=:8080 $OUTPUT_DIR/cpu.prof"
    echo "Then navigate to the 'Flame Graph' view in your browser"
fi

echo ""
echo "Profile files generated in $OUTPUT_DIR/:"
echo "  - cpu.prof: CPU profile (use with 'go tool pprof')"
echo "  - mem.prof: Memory profile"
echo "  - cpu_raw.txt: Raw CPU profile data"
echo ""
echo "To view interactively:"
echo "  go tool pprof -http=:8080 $OUTPUT_DIR/cpu.prof"
