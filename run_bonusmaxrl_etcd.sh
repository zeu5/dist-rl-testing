#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <num_iterations>"
    exit 1
fi

iterations=$1

if [ -d "results" ]; then
    rm -rf results
fi

results_path="results"

./dist-rl-testing etcd cov-rl $iterations --record-event-traces --with-crashes