#!/usr/bin/bash

# Require one argument which is a number
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <num_episodes>"
    exit 1
fi

# Check if the directory exists
if [ -d "results" ]; then
    # Remove the directory
    rm -rf "results"
fi
mkdir results
go build -o tester .

./tester etcd cov --episodes $1

python3 scripts/pure_cov.py results
