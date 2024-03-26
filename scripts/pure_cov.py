import numpy as np
import matplotlib.pyplot as plt
import os
import json

def plot_cov(filepath):
    data = {}
    with open(filepath, 'r') as f:
        data = json.load(f)

    fig, ax = plt.subplots()

    keys = list(data.keys())
    for key in keys:
        timesteps = np.array(data[key]["Timesteps"])
        coverage = np.array(data[key]["UniqueStates"])
        ax.plot(timesteps, coverage, label=key)
    
    ax.legend()
    plt.savefig("results/coverage.png")

if __name__ == "__main__":
    plot_cov("results/0/color_analyzer.json")