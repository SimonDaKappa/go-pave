# Go Benchmark Profiling and Flamegraph Generation Guide

This guide shows you how to profile your Go benchmark tests and generate flamegraphs for performance analysis.

## Quick Start

### Method 1: Interactive Web Interface (Recommended)

```bash
# Run benchmarks with CPU profiling
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof

# Start interactive pprof web server
go tool pprof -http=:8080 cpu.prof
```

Then open your browser to `http://localhost:8080` and click on "Flame Graph" in the top menu.

### Method 2: Using the provided scripts

```bash
# Simple profiling with web interface
./profile_benchmarks.sh

# Advanced flamegraph generation
./generate_flamegraph.sh
```

## Detailed Methods

### 1. Basic CPU Profiling

```bash
# Run all benchmarks with profiling
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof

# Run specific benchmark
go test -bench=BenchmarkParseChainExecution -cpuprofile=cpu.prof

# Run with longer duration for more samples
go test -bench=. -cpuprofile=cpu.prof -benchtime=30s
```

### 2. Memory Profiling

```bash
# Memory allocation profiling
go test -bench=. -memprofile=mem.prof -benchmem

# View memory profile
go tool pprof mem.prof
```

### 3. Viewing Profiles

#### Interactive pprof commands:
```bash
go tool pprof cpu.prof
```

Common commands in pprof:
- `top`: Show top functions by CPU usage
- `top -cum`: Show top functions by cumulative CPU usage
- `web`: Generate SVG graph (requires graphviz)
- `png`: Generate PNG graph
- `list <function>`: Show source code for function
- `help`: Show all commands

#### Web interface:
```bash
go tool pprof -http=:8080 cpu.prof
```

Available views:
- **Flame Graph**: Interactive flamegraph (best for visualization)
- **Graph**: Call graph
- **Top**: List of top functions
- **Source**: Source code view with annotations

### 4. Advanced Profiling Options

```bash
# Block profiling (for goroutine blocking)
go test -bench=. -blockprofile=block.prof

# Mutex profiling (for lock contention)
go test -bench=. -mutexprofile=mutex.prof

# Trace profiling (for detailed execution trace)
go test -bench=. -trace=trace.out
```

### 5. Analyzing Traces

```bash
# View execution trace
go tool trace trace.out
```

## Interpreting Flamegraphs

- **X-axis**: Sample count (wider = more CPU time)
- **Y-axis**: Call stack depth
- **Colors**: Usually random, but consistent for same functions
- **Width**: Proportional to time spent in function
- **Click**: Zoom into specific function calls

## Tips for Better Profiling

1. **Run longer benchmarks** for more samples:
   ```bash
   go test -bench=. -cpuprofile=cpu.prof -benchtime=30s
   ```

2. **Focus on specific benchmarks**:
   ```bash
   go test -bench=BenchmarkParseChainExecution -cpuprofile=cpu.prof
   ```

3. **Use benchmark flags**:
   ```bash
   go test -bench=. -cpuprofile=cpu.prof -benchmem -count=5
   ```

4. **Profile different scenarios**:
   ```bash
   # Profile cached vs uncached performance
   go test -bench=BenchmarkParseChainExecutionCached -cpuprofile=cached.prof
   go test -bench=BenchmarkParseChainExecution -cpuprofile=uncached.prof
   ```

## Common Performance Issues to Look For

1. **Hot paths**: Functions that appear wide in flamegraph
2. **Deep call stacks**: Might indicate inefficient algorithms
3. **Memory allocations**: Use `-benchmem` and memory profiles
4. **Lock contention**: Use mutex profiling
5. **GC pressure**: Look for runtime.gc* functions in profiles

## Your Current Benchmarks

You have two benchmark functions:
- `BenchmarkParseChainExecution`: Tests parsing without caching
- `BenchmarkParseChainExecutionCached`: Tests parsing with pre-cached chains

Compare their profiles to see the impact of caching on performance.

## Example Analysis Commands

```bash
# Compare two profiles
go tool pprof -base=cached.prof uncached.prof

# Generate comparison flamegraph
go tool pprof -http=:8080 -base=cached.prof uncached.prof
```
