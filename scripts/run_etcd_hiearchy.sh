#!/usr/bin/bash

# Require one argument which is a number
if [ "$#" -lt 2 ]; then
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

if [ "$#" -gt 2 ]; then 
    ./tester etcd hierarchy $1 --episodes $2 ${@:3}
else 
    ./tester etcd hierarchy $1 --episodes $2
fi

python3 scripts/hierarchy_cov.py results
