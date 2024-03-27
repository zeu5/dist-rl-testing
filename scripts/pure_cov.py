import numpy as np
import matplotlib.pyplot as plt
import os
import json
import sys

def plot_cov(dirpath):
    data = []
    for run in os.listdir(dirpath):
        if os.path.isdir(os.path.join(dirpath, run)):
            with open(os.path.join(dirpath, run, "color_analyzer.json"), 'r') as f:
                data.append(json.load(f))
    
    fig, ax = plt.subplots()

    keys = list(data[0].keys())
    for key in keys:
        timesteps = np.array(data[0][key]["Timesteps"])
        coverage = np.array(data[0][key]["UniqueStates"])
        ax.plot(timesteps, coverage, label=key)
    
    ax.legend()
    plt.savefig("results/coverage.png")

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python pure_cov.py <path_to_directory>")
        sys.exit(0)

    plot_cov(sys.argv[1])