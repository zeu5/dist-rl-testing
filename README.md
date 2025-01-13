# dist-rl-testing

RL testing for distributed systems. This repository contains an implementation of a new RL based exploration algorithm and benchmarks (etcd and RSL) to evaluate the new exploration algorithm.

- etcd: the popular key value store
- RSL: the algorithm used inside Azure storage

## Directory structure

- `benchmarks` - contains the code to instantiate and run the different algorithms based on the command line parameters provided.
- `core` - contains the main interfaces, implementations of the generic environments and login run different experiments (in parallel).
- `policies` - implementations of the different algorithms to benchmark on.
- `analysis` - code to perform the post-processing of the experiments to generate graphs and data.
- `scripts` - Easy scripts to run the different benchmarks
- `util` - auxilliary libraries

## Setup instructions

To build the docker image,

``` bash
docker build -t dist-rl-testing .
```

To run the docker image,

``` bash
docker run -it dist-rl-testing:latest
```

The scripts will open an interactive shell to run further experiments

## Step-by-Step instructions to run the experiments

Once the docker has been instantiated, run the following instructions to reproduce results from the table. The results are written to the `results` folder.

### Kick in the tyres phase

``` bash
bash scripts/run_cov.sh etcd 1000
```

### Full evaluation

For RQ1

```bash
bash scripts/run_cov.sh etcd 10000 --num-runs 10
```

For RQ2

```bash
bash scripts/run_hierarchy.sh etcd set1 10000 --num-runs 10
```

Replace `etcd` with `rsl` for the RSL benchmark.
