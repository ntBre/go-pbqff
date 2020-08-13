package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var (
	arrs   = []*[]CountFloat{&e2d, &fc2, &fc3, &fc4}
	fnames = []string{"e2d.json", "fc2.json", "fc3.json", "fc4.json"}
)

func MakeCheckpoint() {
	for a, arr := range arrs {
		temp := make([]CountFloat, 0, len(*arrs[a]))
		for _, v := range *arr {
			if v.Done() {
				v.Loaded = true
				temp = append(temp, v)
			} else {
				temp = append(temp, CountFloat{})
			}
		}
		aJSON, err := json.MarshalIndent(temp, "", "\t")
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(fnames[a], aJSON, 0755)
	}
}

func LoadCheckpoint() {
	for a := range arrs {
		lines, _ := ioutil.ReadFile(fnames[a])
		err := json.Unmarshal(lines, &arrs[a])
		if err != nil {
			errExit(err, fmt.Sprintf("loading %s for checkpoint", fnames[a]))
		}
	}
}
