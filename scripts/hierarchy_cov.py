import numpy as np
import matplotlib.pyplot as plt
import pandas as pd
import os
import json
import sys
from pathlib import Path

def plot_cov(dirpath):
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
                if pred not in data:
                    data[pred] = []
                with open(os.path.join(run_dir, file_name), 'r') as pred_file:
                    data[pred].append(json.load(pred_file))

    avg_data = {
        "Predicate": [],
        "PredHRL": [],
        "BonusMax": [],
        "NegRLVisits": [],
        "Random": []
    }

    for pred, pred_runs in data.items():
        fig, ax = plt.subplots()

        avg_data["Predicate"].append(pred)

        for key in ["PredHRL_"+pred, "Random", "BonusMax", "NegRLVisits"]:
            min_len = min([len(pred_runs[i][key]["FinalPredicateStates"]) for i in range(len(pred_runs))])
            filtered_runs = [pred_runs[i][key]["FinalPredicateStates"][:min_len] for i in range(len(pred_runs))]
            
            timesteps = pred_runs[0][key]["FinalPredicateTimesteps"][:min_len]


            mean = np.mean(filtered_runs, axis=0)
            std = np.std(filtered_runs, axis=0)
            ax.plot(timesteps, mean, label=key)
            ax.fill_between(timesteps, mean+std, mean-std, alpha=0.2)

            avg_data_key = key
            if "PredHRL" in key:
                avg_data_key = "PredHRL"
            avg_data[avg_data_key].append((mean[-1], std[-1]))
        
        ax.legend()
        ax.set_title("Predicate {} analysis".format(pred))
        plt.savefig("{}/{}_coverage.png".format(dirpath, pred))
    
    df = pd.DataFrame(avg_data, index=avg_data["Predicate"], columns=["PredHRL", "BonusMax", "NegRLVisits", "Random"])
    df.to_csv("{}/avg_coverage.csv".format(dirpath))

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python pure_cov.py <path_to_directory>")
        sys.exit(0)

    plot_cov(sys.argv[1])