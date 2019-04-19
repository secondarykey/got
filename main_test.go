package main

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestIgnore(t *testing.T) {
	flag := Ignore(fsnotify.Event{Name: "main.go"})
	if flag {
		t.Error("main.go not ignore")
	}
	flag = Ignore(fsnotify.Event{Name: "notexists.go"})
	if !flag {
		t.Error("notexists.go ignore")
	}
	flag = Ignore(fsnotify.Event{Name: "go.mod"})
	if !flag {
		t.Error("go.mod exist but go file")
	}
}
