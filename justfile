# justfile for PG&E Continuous Running Cost Estimator

# Default action: list available commands
default:
    @just --list

# Build the pge-cost executable binary
build:
    go build -o pge-cost main.go

# Run the cost estimator with default parameters (150W on E-TOU-C)
run plan="E-TOU-C" watts="150": build
    ./pge-cost --watts {{watts}} --plan {{plan}}

# Run cost estimator for tiered E-1 plan
run-tiered watts="150" tier="2": build
    ./pge-cost --watts {{watts}} --plan E-1 --tier {{tier}}

# Run all unit tests
test:
    go test -v ./...

# Run static analysis and Go vetting checks
vet:
    go vet ./...

# Format all Go source files
fmt:
    go fmt ./...

# Fetch and update rates database directly from PG&E's website
fetch db="rates.json": build
    ./pge-cost fetch --db {{db}}

# Clean up built binaries and downloaded spreadsheets
clean:
    rm -f pge-cost
    rm -rf .tmp/downloaded-rates.xlsx
