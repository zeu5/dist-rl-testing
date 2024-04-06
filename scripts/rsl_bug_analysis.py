import os
import sys
import numpy as np
import pandas as pd

def compare_bugs(bugs_dir):
    data = {}
    
    for file in os.listdir(bugs_dir):
        parts = file.split(".")[0].split("_")
        run = parts[0]
        algo = ""
        bug = ""
        episode = parts[-1]
        if len(parts) == 6:
            algo = "_".join(parts[1:3])
            bug = parts[3]
        else:
            algo = parts[1]
            bug = parts[2]

        if (algo, bug) not in data:
            data[(algo, bug)] = []
        data[(algo, bug)].append((run, episode))
    
    avg_occurrence = {

    }
    occurrences = {
        "Bugs": []
    }
    for (algo, bug) in data:
        if algo not in occurrences:
            occurrences[algo] = []
        if bug not in occurrences["Bugs"]:
            occurrences["Bugs"].append(bug)
        
        run_occurs = {}
        for (run, episode) in data[(algo, bug)]:
            if run not in run_occurs:
                run_occurs[run] = 0
            run_occurs[run] += 1
        
        avg_occurrence[(algo, bug)] = np.mean(list(run_occurs.values()))
    
    num_bugs = len(occurrences["Bugs"])
    for (algo, bug) in avg_occurrence:
        occurrences[algo] = [0]*num_bugs
        bug_index = occurrences["Bugs"].index(bug)
        occurrences[algo][bug_index] = avg_occurrence[(algo, bug)]
    
    df = pd.DataFrame(occurrences)
    print(df)

if __name__ == "__main__":

    if len(sys.argv) != 2:
        print("usage: redis_bugs.py <analysis_dir>")
        exit(1)
    
    root_dir = sys.argv[1]
    if not os.path.isdir(root_dir):
        print("specified path not a directory!")
        exit(1)
    
    compare_bugs(sys.argv[1])
