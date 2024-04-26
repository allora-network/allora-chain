# SPDX-License-Identifier: AGPL-3.0

## First install the following tools:
# go install github.com/jandelgado/gcov2lcov@latest 
# brew install lcov


# if any command fails then immediately exit
set -o errexit

# 1) test
go test ./... -coverprofile=coverage.out

# 2) convert result to lcov
gcov2lcov -infile=coverage.out -outfile=coverage_raw.lcov

exclude_patterns=(
    'x/emissions/types/*.pb.go'
    'x/emissions/types/*.pb.gw.go'
    'math/collections.go'
)

# 3) Exclude files from coverage results
lcov --remove coverage_raw.lcov "${exclude_patterns[@]}" -o coverage.lcov
# lcov --remove coverage_raw.lcov 'x/emissions/types/*.pb.go' -o coverage.lcov

rm coverage_raw.lcov

# 4) Clean coverage directory
rm -rf coverage
mkdir -p coverage

# 5) create html viewable coverage report
genhtml coverage.lcov --dark-mode -o coverage

# 6) cleanup
rm -f coverage/coverage.lcov
mv coverage.lcov coverage

echo 'Success! Coverage data viewable in coverage/index.html'
