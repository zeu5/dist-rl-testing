import numpy as np
import matplotlib.pyplot as plt
import os
import json
import sys

def plot_cov(dirpath):
    runs = []
    for run in os.listdir(dirpath):
        if not run.isdigit():
            continue
        if os.path.isdir(os.path.join(dirpath, run)):
            with open(os.path.join(dirpath, run, "color_analyzer.json"), 'r') as f:
                runs.append(json.load(f))
    
    fig, ax = plt.subplots()

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
        ax.plot(timesteps, mean, label=key)
        ax.fill_between(timesteps, mean+std, mean-std, alpha=0.2)
    
    ax.legend()
    plt.savefig("{}/coverage.png".format(dirpath))

    for key in data:
        min_len = min([len(run) for run in data[key]])
        filtered_data = [run[:min_len] for run in data[key]]
        mean = np.mean(filtered_data, axis=0)
        std = np.std(filtered_data, axis=0)
        print("Mean final coverage using {}: {}+/-{}".format(key, mean[-1], std[-1]))

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python pure_cov.py <path_to_directory>")
        sys.exit(0)

    plot_cov(sys.argv[1])