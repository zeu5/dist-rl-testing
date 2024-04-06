import numpy as np
import matplotlib.pyplot as plt
import pandas as pd
import os
import json
import sys
from pathlib import Path
from scipy.stats import mannwhitneyu

def plot_cov(dirpath, final_pred):
    data = {}
    for run in os.listdir(dirpath):
        if not run.isdigit():
            continue
        if os.path.isdir(os.path.join(dirpath, run)):
            run_dir = os.path.join(dirpath, run)
            predicate_files = {}
            for file in os.listdir(run_dir):
                if "predicate_comparison_" in file:
                    pred = "_".join(file.split(".")[0].split("_")[2:])
                    predicate_files[pred] = file
            
            for pred, file_name in predicate_files.items():
                if pred != final_pred:
                    continue
                if pred not in data:
                    data[pred] = []
                with open(os.path.join(run_dir, file_name), 'r') as pred_file:
                    data[pred].append(json.load(pred_file))

    fig, ax = plt.subplots()

    algos = list(data[final_pred][0].keys())
    runs = data[final_pred]

    for algo in algos:
        min_len = len(runs[0][algo]["FinalPredicateStates"])
        for i in range(len(runs)):
            if algo not in runs[i]:
                continue
            l = len(runs[i][algo]["FinalPredicateStates"])
            if l < min_len:
                min_len = l
        filtered_runs = []
        timesteps = []
        for i in range(len(runs)):
            if algo not in runs[i]:
                continue
            filtered_runs.append(runs[i][algo]["FinalPredicateStates"][:min_len])
            if len(timesteps) == 0:
                timesteps = runs[i][algo]["FinalPredicateTimesteps"][:min_len]

        mean = np.mean(filtered_runs, axis=0)
        std = np.std(filtered_runs, axis=0)
        ax.plot(timesteps, mean, label=algo)
        ax.fill_between(timesteps, mean+std, mean-std, alpha=0.2)
    
        ax.legend()
        ax.set_title("Predicate {} analysis".format(final_pred))
        plt.savefig("{}/{}_coverage.png".format(dirpath, final_pred))

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python pure_cov.py <path_to_directory> <predicate>")
        sys.exit(0)

    plot_cov(sys.argv[1], sys.argv[2])