#!/bin/bash

# Script to profile Go benchmarks and generate flamegraphs
# Usage: ./profile_benchmarks.sh [benchmark_pattern]

set -e

BENCHMARK_PATTERN=${1:-"."}
OUTPUT_DIR="profiles"

echo "Creating output directory..."
mkdir -p $OUTPUT_DIR

echo "Running benchmarks with profiling..."
go test -bench=$BENCHMARK_PATTERN -cpuprofile=$OUTPUT_DIR/cpu.prof -memprofile=$OUTPUT_DIR/mem.prof -benchtime=10s

echo "Starting pprof web server for CPU profile..."
echo "Open your browser to http://localhost:8080 to view the profile"
echo "In the web interface, click on 'Flame Graph' in the top menu to see the flamegraph"
echo ""
echo "Press Ctrl+C to stop the server"

go tool pprof -http=:8080 $OUTPUT_DIR/cpu.prof
