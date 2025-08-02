# Example Tests

This directory contains test files for each zlog example to ensure they compile and function correctly.

## Test Files

Each example has a corresponding `main_test.go` file that includes:

### 1. Build Tests
- `TestBuild<ExampleName>`: Ensures the example compiles without errors
- Verifies all dependencies are properly resolved
- Cleans up generated binaries

### 2. Run Tests
- `TestRun<ExampleName>`: Executes the example and verifies expected behavior
- Checks for expected output sections and messages
- Validates that key features are demonstrated
- Uses timeouts to prevent hanging tests
- Skipped in short mode (`go test -short`)

### 3. Feature Tests
- Additional tests that verify specific features demonstrated in each example
- Ensures all expected components and patterns are present

## Running Tests

### Test Individual Example
```bash
cd custom-fields
go test -v
```

### Test All Examples (Quick)
```bash
# From examples directory
./test_all_examples.sh
```

### Test All Examples (Full)
```bash
# From examples directory
for dir in */; do
    if [ -f "$dir/main_test.go" ]; then
        echo "Testing $dir"
        (cd "$dir" && go test -v)
    fi
done
```

## Test Coverage

The tests verify:

1. **custom-fields**:
   - Field transformation functions (redaction, masking, hashing)
   - Security and compliance features
   - All output sections are present

2. **custom-signals**:
   - Custom signal definitions and routing
   - Sink behaviors (audit, metrics, analytics, alerts)
   - Context propagation and trace IDs
   - Sampling functionality

3. **custom-sink**:
   - Different sink patterns (metrics, message queue, database, conditional, batching)
   - Sink outputs and behaviors
   - Metrics aggregation

4. **event-pipeline**:
   - Complete event pipeline setup
   - Session correlation
   - Multiple sink coordination
   - Business analytics and alerting

5. **resilient-sinks**:
   - Circuit breaker state changes and fallback
   - Rate limiting behavior
   - Combined protection patterns
   - Performance under load

6. **standard-logging**:
   - Traditional log levels (DEBUG, INFO, WARN, ERROR)
   - JSON structured output
   - RouteAll functionality
   - Custom signals alongside standard logging

## CI/CD Integration

These tests can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
test-examples:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    - name: Test Examples
      run: |
        cd examples
        ./test_all_examples.sh
```

## Notes

- Tests use `-short` flag to skip execution tests in CI environments
- All tests clean up generated files (binaries, log files)
- Tests are designed to be idempotent and can be run multiple times
- Timeouts prevent tests from hanging on unexpected behavior