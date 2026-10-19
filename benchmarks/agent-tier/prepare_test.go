package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestPreparedScheduleRandomizesBalancedFirstAssignments(t *testing.T) {
	// Catches assigning first condition by original question/repetition parity
	// with only a global flip, rather than randomizing balanced assignments.
	root := t.TempDir()
	set := "../../internal/benchmarks/testdata/fixture-tiny"
	private := filepath.Join(root, "private")
	if data, err := invoke(t, "stub", "--set", set, "--out", private); err != nil {
		t.Fatalf("%v: %s", err, data)
	}
	p := readObject(t, filepath.Join(private, "protocol.json"))
	p["repetitions"] = 8
	assignments := map[string]bool{}
	for seed := 0; seed < 8; seed++ {
		p["run_order_seed"] = seed
		data, _ := json.Marshal(p)
		file := filepath.Join(root, "protocol.json")
		if err := os.WriteFile(file, data, 0600); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(root, string(rune('a'+seed)))
		if data, err := invoke(t, "prepare", "--set", set, "--protocol", file, "--out", out); err != nil {
			t.Fatalf("%v: %s", err, data)
		}
		data, err := os.ReadFile(filepath.Join(out, "schedule.json"))
		if err != nil {
			t.Fatal(err)
		}
		var slots []Slot
		if err = json.Unmarshal(data, &slots); err != nil {
			t.Fatal(err)
		}
		first := []string{}
		count := 0
		for i := 0; i < len(slots); i += 2 {
			a, c := slots[i], slots[i+1]
			if a.QuestionID != c.QuestionID || a.Repetition != c.Repetition || a.Condition == c.Condition {
				t.Fatal("broken adjacent pair")
			}
			if a.Condition == "with_rforge" {
				count++
			}
			first = append(first, string(rune('a'+a.Repetition))+a.Condition)
		}
		if count != 4 {
			t.Fatalf("unbalanced order: %d", count)
		}
		sort.Strings(first)
		assignments[strings.Join(first, ",")] = true
	}
	if len(assignments) <= 2 {
		t.Fatalf("only %d first-condition assignments across seeds; fixed parity confounds repetition", len(assignments))
	}
}
