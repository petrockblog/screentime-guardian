# Test Coverage

Comprehensive test suite for the parental control daemon.

## Test Structure

Tests follow Go conventions:
- Test files are named `*_test.go`
- Tests are in the same package as the code they test
- Mock implementations are in separate files (`mock.go`)

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/config
go test ./internal/storage
go test ./internal/scheduler

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Packages

### config
- `config_test.go`: Tests configuration loading, saving, and defaults

### storage
- `storage_test.go`: Tests database operations (users, usage tracking, time extensions)

### dbus
- `mock.go`: Mock implementations of LogindClient and Notifier for testing
- Tests verify session management mocking works correctly

### notifier
- `mock.go`: Mock Notifier implementation
- `notifier_test.go`: Tests notification chain pattern

### scheduler
- `scheduler_test.go`: Tests time tracking, warnings, and lock enforcement using mocks

## Mocking Strategy

D-Bus operations and notifications cannot run on macOS, so we provide mock implementations:
- `dbus.MockLogindClient`: Simulates session locking/termination
- `dbus.MockNotifier`: Simulates desktop notifications  
- `notifier.MockNotifier`: Test implementation of Notifier interface

These mocks are used in scheduler tests to verify logic without requiring Linux.

## Coverage Goals

- **Config**: 100% (simple serialization logic)
- **Storage**: >90% (core business logic)
- **Notifier**: >80% (chain pattern + mocks)
- **Scheduler**: >75% (complex time logic with mocks)
- **API**: Not yet tested (future: httptest)
