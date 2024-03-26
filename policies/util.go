package policies

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

type QTable struct {
	table map[string]map[string]float64

	rand *rand.Rand
}

func NewQTable() *QTable {
	return &QTable{
		table: make(map[string]map[string]float64),
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (q *QTable) GetAll(state string) (map[string]float64, bool) {
	values, ok := q.table[state]
	return values, ok
}

func (q *QTable) Get(state, action string, def float64) float64 {
	if _, ok := q.table[state]; !ok {
		q.table[state] = make(map[string]float64)
	}
	if _, ok := q.table[state][action]; !ok {
		q.table[state][action] = def
	}
	return q.table[state][action]
}

func (q *QTable) Set(state, action string, val float64) {
	if _, ok := q.table[state]; !ok {
		q.table[state] = make(map[string]float64)
	}
	q.table[state][action] = val
}

func (q *QTable) HasState(state string) bool {
	_, ok := q.table[state]
	return ok
}

func (q *QTable) Max(state string, def float64) (string, float64) {
	if _, ok := q.table[state]; !ok { // if state is not in the QTable at all, return default value
		q.table[state] = make(map[string]float64)
		return "", def
	}
	maxAction := ""
	maxVal := float64(math.MinInt)
	for a, val := range q.table[state] { // for all the actions
		if val > maxVal {
			maxAction = a
			maxVal = val
		}
	}

	if maxAction == "" {
		return "", def
	}

	return maxAction, maxVal
}

func (q *QTable) Exists(state string) bool {
	_, ok := q.table[state]
	return ok
}

func (q *QTable) Size() int {
	return len(q.table)
}

func (q *QTable) MaxAmong(state string, actions []string, def float64) (string, float64) {
	if _, ok := q.table[state]; !ok {
		q.table[state] = make(map[string]float64)
	}
	maxActions := make([]string, 0)
	maxVal := float64(math.MinInt)
	for _, a := range actions {
		if _, ok := q.table[state][a]; !ok {
			q.table[state][a] = def
		}
		val := q.table[state][a]
		if val > maxVal {
			maxActions = make([]string, 0)
			maxVal = val
		}
		if val == maxVal {
			maxActions = append(maxActions, a)
		}
	}

	randAction := q.rand.Intn(len(maxActions))
	return maxActions[randAction], maxVal
}

func (q *QTable) Read(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		in := make(map[string]interface{})
		err := json.Unmarshal(scanner.Bytes(), &in)
		if err != nil {
			return fmt.Errorf("error reading file contents: %s", err)
		}
		state := in["state"].(string)
		entries := make(map[string]float64)
		for k, v := range in["entries"].(map[string]interface{}) {
			entries[k] = v.(float64)
		}
		q.table[state] = entries
	}
	return nil
}

func (q *QTable) Record(path string) {
	bs := new(bytes.Buffer)

	for state, entries := range q.table {
		stateJ := make(map[string]interface{})
		stateJ["state"] = state
		stateJ["entries"] = entries

		stateBS, err := json.Marshal(stateJ)
		if err == nil {
			bs.Write(stateBS)
			bs.Write([]byte("\n"))
		}
	}

	if bs.Len() > 0 {
		os.WriteFile(path+".jsonl", bs.Bytes(), 0644)
	}
}
