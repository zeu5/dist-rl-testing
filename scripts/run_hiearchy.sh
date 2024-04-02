#!/usr/bin/bash

# Require one argument which is a number
if [ "$#" -lt 3 ]; then
    echo "Usage: $0 <benchmark> <hierarchy> <num_episodes>"
    exit 1
fi

# Check if the directory exists
if [ -d "results" ]; then
    # Remove the directory
    rm -rf "results"
fi
mkdir results
go build -o tester .

if [ "$#" -gt 3 ]; then 
    ./tester $1 hierarchy $2 --episodes $3 ${@:4}
else 
    ./tester $1 hierarchy $2 --episodes $3
fi

python3 scripts/hierarchy_cov.py results
