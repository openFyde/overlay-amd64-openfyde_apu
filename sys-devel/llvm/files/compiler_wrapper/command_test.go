// Copyright 2019 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestMergeEnvValues(t *testing.T) {
	testData := []struct {
		values  []string
		updates []string
		result  []string
	}{
		{[]string{}, []string{}, []string{}},
		{[]string{"A=1"}, []string{}, []string{"A=1"}},
		{[]string{"A=1=2=3"}, []string{}, []string{"A=1=2=3"}},
		{[]string{}, []string{"A=1"}, []string{"A=1"}},
		{[]string{}, []string{"A=1=2=3"}, []string{"A=1=2=3"}},
		{[]string{"A=1"}, []string{"A=2"}, []string{"A=2"}},
		{[]string{"A="}, []string{}, []string{"A="}},
		{[]string{"A="}, []string{"A=2"}, []string{"A=2"}},
		{[]string{"A=1"}, []string{"A="}, []string{}},
		{[]string{}, []string{"A=1", "A="}, []string{}},
		{[]string{}, []string{"A=1", "A=", "A=2"}, []string{"A=2"}},
		{[]string{"A=1", "B=2"}, []string{"C=3", "D=4"}, []string{"A=1", "B=2", "C=3", "D=4"}},
	}
	for _, tt := range testData {
		result := mergeEnvValues(tt.values, tt.updates)
		sort.Strings(result)
		if !reflect.DeepEqual(tt.result, result) {
			t.Errorf("unexpected result: %s", result)
		}
	}
}
