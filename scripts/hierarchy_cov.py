import numpy as np
import matplotlib.pyplot as plt
import os
import json
import sys
from pathlib import Path

def plot_cov(dirpath):
    data = {}
    for run in os.listdir(dirpath):
        if os.path.isdir(os.path.join(dirpath, run)):
            run_dir = os.path.join(dirpath, run)
            predicate_files = {}
            for file in os.listdir(run_dir):
                if "predicate_comparison_" in file:
                    pred = "_".join(file.split(".")[0].split("_")[2:])
                    predicate_files[pred] = file
            
            for pred, file_name in predicate_files.items():
                data[pred] = []
                with open(os.path.join(run_dir, file_name), 'r') as pred_file:
                    data[pred].append(json.load(pred_file))

    for pred, pred_runs in data.items():
        fig, ax = plt.subplots()

        keys = list(pred_runs[0].keys())
        for key in keys:
            timesteps = np.array(pred_runs[0][key]["FinalPredicateTimesteps"])
            coverage = np.array(pred_runs[0][key]["FinalPredicateStates"])
            ax.plot(timesteps, coverage, label=key)
        
        ax.legend()
        ax.set_title("Predicate {} analysis".format(pred))
        plt.savefig("results/{}_coverage.png".format(pred))

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python pure_cov.py <path_to_directory>")
        sys.exit(0)

    plot_cov(sys.argv[1])