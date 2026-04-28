#!/bin/bash

# ==========================================
# QIM Admin Frontend Automated Test Runner
# ==========================================
# Usage:
#   ./run-tests.sh          # Run all tests
#   ./run-tests.sh coverage # Run tests with coverage report
#   ./run-tests.sh watch    # Run tests in watch mode
#   ./run-tests.sh ci       # Run tests in CI mode (with JUnit report)
# ==========================================

set -e

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}  QIM Admin Frontend Test Runner${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Check if dependencies are installed
if [ ! -d "$PROJECT_ROOT/node_modules" ]; then
  echo -e "${YELLOW}Installing dependencies...${NC}"
  cd "$PROJECT_ROOT" && npm install
  echo ""
fi

# Function to run tests
run_tests() {
  echo -e "${BLUE}Running tests...${NC}"
  echo ""
  cd "$PROJECT_ROOT" && npm test
  local exit_code=$?
  echo ""
  if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
  else
    echo -e "${RED}Some tests failed!${NC}"
  fi
  return $exit_code
}

# Function to run coverage
run_coverage() {
  echo -e "${BLUE}Running tests with coverage...${NC}"
  echo ""
  cd "$PROJECT_ROOT" && npm run test:coverage
  local exit_code=$?
  echo ""
  if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}Coverage report generated!${NC}"
    echo -e "${BLUE}HTML report: $PROJECT_ROOT/coverage/index.html${NC}"
  else
    echo -e "${RED}Coverage failed!${NC}"
  fi
  return $exit_code
}

# Function to run watch mode
run_watch() {
  echo -e "${BLUE}Running tests in watch mode...${NC}"
  echo ""
  cd "$PROJECT_ROOT" && npm run test:watch
}

# Function to run CI mode
run_ci() {
  echo -e "${BLUE}Running tests in CI mode...${NC}"
  echo ""

  # Create output directory
  mkdir -p "$PROJECT_ROOT/test-results"

  cd "$PROJECT_ROOT" && npx vitest run --reporter=junit --outputFile=test-results/junit.xml 2>&1 || true
  cd "$PROJECT_ROOT" && npm run test:coverage -- --reporter=json --outputFile=test-results/coverage.json 2>&1 || true

  echo ""
  echo -e "${BLUE}Test results saved to: $PROJECT_ROOT/test-results/${NC}"
}

# Function to run specific test file
run_specific() {
  local test_file="$1"
  echo -e "${BLUE}Running specific test: ${test_file}${NC}"
  echo ""
  cd "$PROJECT_ROOT" && npx vitest run "$test_file"
}

# Main logic
case "${1:-}" in
  coverage)
    run_coverage
    ;;
  watch)
    run_watch
    ;;
  ci)
    run_ci
    ;;
  specific)
    run_specific "${2:-}"
    ;;
  *)
    run_tests
    ;;
esac
