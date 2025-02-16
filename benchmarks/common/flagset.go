package common

import (
	"path"
	"time"

	"github.com/zeu5/dist-rl-testing/util"
)

type Flags struct {
	PartitionFlags
	SavePath string
	RunFlags
	Parallelism       int
	Debug             bool
	RecordEventTraces bool
}

type PartitionFlags struct {
	NumNodes              int
	Requests              int
	TicksBetweenPartition int
	StaySameUpto          int
	WithCrashes           bool
	MaxCrashedNodes       int
	MaxCrashActions       int
}

type RunFlags struct {
	NumRuns                int
	Episodes               int
	Horizon                int
	MaxConsecutiveErrors   int
	MaxConsecutiveTimeouts int
	EpisodeTimeout         time.Duration
}

func DefaultFlags() *Flags {
	return &Flags{
		PartitionFlags: PartitionFlags{
			NumNodes:              3,
			Requests:              5,
			TicksBetweenPartition: 4,
			StaySameUpto:          5,
			WithCrashes:           true,
			MaxCrashedNodes:       1,
			MaxCrashActions:       3,
		},
		SavePath: "results",
		RunFlags: RunFlags{
			NumRuns:                1,
			Episodes:               1000,
			Horizon:                25,
			MaxConsecutiveErrors:   20,
			MaxConsecutiveTimeouts: 20,
			EpisodeTimeout:         10 * time.Second,
		},
		Parallelism:       10,
		Debug:             false,
		RecordEventTraces: false,
	}
}

func (f *Flags) Record() {
	util.SaveJson(path.Join(f.SavePath, "config.json"), f)
}
