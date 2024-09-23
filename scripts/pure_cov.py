import numpy as np
import matplotlib.pyplot as plt
import os
import json
import sys
from scipy.stats import mannwhitneyu
import pandas as pd

color_keys = {
    "BonusMax": "red",
    "NegRLVisits": "blue",
    "UCBZero": "blue",
    "Random": "green"
}

label_keys = {
    "BonusMax": "BonusMaxRL",
    "NegRLVisits": "NegRLVisits",
    "Random": "Random",
    "UCBZero": "UCBZero",
}

style_keys = {
    "BonusMax": "dotted",
    "NegRLVisits": "dashed",
    "UCBZero": "dashed",
    "Random": "dashdot"
}

def plot_cov(dirpath):
    runs = []
    for run in os.listdir(dirpath):
        if not run.isdigit():
            continue
        if os.path.isdir(os.path.join(dirpath, run)):
            with open(os.path.join(dirpath, run, "color_analyzer.json"), 'r') as f:
                runs.append(json.load(f))
    
    fig, ax = plt.subplots()
    plt.xlabel("Time steps")
    plt.ylabel("Unique states")

    data = {}
    for r in runs:
        for key in r:
            if key not in data:
                data[key] = []
            data[key].append(r[key]["UniqueStates"])


    for key in data:
        min_len = min([len(run) for run in data[key]])
        filtered_data = [run[:min_len] for run in data[key]]
        timesteps = runs[0][key]["Timesteps"][:min_len]

        mean = np.mean(filtered_data, axis=0)
        std = np.std(filtered_data, axis=0)
        ax.plot(timesteps, mean, label=label_keys[key], color=color_keys[key], linestyle=style_keys[key])
        ax.fill_between(timesteps, mean+std, mean-std, color=color_keys[key], alpha=0.2)
    
    ax.legend()
    plt.savefig("{}/coverage.pdf".format(dirpath),  bbox_inches="tight")

    for key in data:
        min_len = min([len(run) for run in data[key]])
        filtered_data = [run[:min_len] for run in data[key]]
        mean = np.mean(filtered_data, axis=0)
        std = np.std(filtered_data, axis=0)
        print("Mean final coverage using {}: {}+/-{}".format(key, mean[-1], std[-1]))

    state_test = {}
    for key1 in data:
        key1_final_coverages = [run[-1] for run in data[key1]]
        state_test[key1] = {}
        for key2 in data:
            if key1 == key2:
                continue
            key2_final_coverages = [run[-1] for run in data[key2]]

            stat, p = mannwhitneyu(key1_final_coverages, key2_final_coverages, alternative="greater")
            state_test[key1][key2] = (stat, p)
    with open("{}/cov_stat.json".format(dirpath), "w") as stat_file:
        json.dump(state_test, stat_file)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python pure_cov.py <path_to_directory>")
        sys.exit(0)

    plot_cov(sys.argv[1])