package main

import "testing"

func TestLocal(t *testing.T) {
	testCases := []struct {
		rawURL string
		expPos [2]int
	}{{
		rawURL: "https://foobar.com#L1:23",
		expPos: [2]int{1, 23},
	}, {
		rawURL: "https://foobar.com#L2:8",
		expPos: [2]int{2, 8},
	}, {
		rawURL: "https://foobar.com#L2",
		expPos: [2]int{2, 0},
	}, {
		rawURL: "https://foobar.com",
		expPos: [2]int{0, 0},
	}}

	for _, testCase := range testCases {
		line, col, err := localPosition(testCase.rawURL)
		if err != nil {
			t.Fatal(err)
		}
		if testCase.expPos[0] != line || testCase.expPos[1] != col {
			t.Errorf("line and/or column do not match %+v != %+v", testCase.expPos, [2]int{line, col})
		}
	}
}
