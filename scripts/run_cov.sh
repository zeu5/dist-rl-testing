#!/usr/bin/bash

# Require one argument which is a number
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <benchmark> <num_episodes>"
    exit 1
fi

# Check if the directory exists
if [ -d "results" ]; then
    # Remove the directory
    rm -rf "results"
fi
mkdir results
go build -o tester .

if [ "$#" -gt 2 ]; then 
    ./tester $1 cov --episodes $2 ${@:3}
else 
    ./tester $1 cov --episodes $2
fi

source venv/bin/activate
python3 scripts/pure_cov.py results
deactivate
