#!/usr/bin/bash

# Require one argument which is a number
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <hierarchy> <num_episodes>"
    exit 1
fi

# Check if the directory exists
if [ -d "results" ]; then
    # Remove the directory
    rm -rf "results"
fi
mkdir results
go build -o tester .

./tester etcd hierarchy $1 --episodes $2

python3 scripts/hierarchy_cov.py results
