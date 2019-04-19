package main

import (
	"testing"
)

func TestIgnore(t *testing.T) {
	flag := Ignore("main.go")
	if flag {
		t.Error("main.go not ignore")
	}
	flag = Ignore("notexists.go")
	if !flag {
		t.Error("notexists.go ignore")
	}
	flag = Ignore("go.mod")
	if !flag {
		t.Error("go.mod exist but go file")
	}
}
