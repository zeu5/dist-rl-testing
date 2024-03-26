#!/usr/bin/bash

# Check if the directory exists
if [ -d "results" ]; then
    # Remove the directory
    rm -rf "results"
fi
mkdir results
go build -o tester .

./tester etcd cov

python3 scripts/pure_cov.py 
