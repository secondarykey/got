package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const Duration = 5

var (
	args        []string
	defaultArgs []string
)

func init() {
	defaultArgs = []string{"test", "-v", "./..."}
}

func main() {

	if len(os.Args) > 1 {
		args = make([]string, len(os.Args))
		args[0] = "test"
		for idx, elm := range os.Args[1:] {
			args[idx+1] = elm
		}

	} else {
		args = defaultArgs
	}

	err := circuit()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

var lastUnixTime int64

func lock() bool {
	now := time.Now().Unix()
	if now > lastUnixTime+Duration {
		return true
	}
	return false
}

func unlock() {
	now := time.Now()
	lastUnixTime = now.Unix()
}

func circuit() error {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	watcher.Add("./")
	if err != nil {
		return nil
	}

	done := make(chan error)
	ci := make(chan bool)

	defer close(ci)
	defer close(done)

	go func() {
		for {
			select {
			case event := <-watcher.Events:

				switch {
				case event.Op&fsnotify.Rename == fsnotify.Rename:
				case event.Op&fsnotify.Create == fsnotify.Create:
				case event.Op&fsnotify.Remove == fsnotify.Remove:
				case event.Op&fsnotify.Chmod == fsnotify.Chmod:
				case event.Op&fsnotify.Write == fsnotify.Write:
				default:
				}

				if !Ignore(event.Name) {
					ci <- true
				}

			case err := <-watcher.Errors:
				done <- err
			}
		}
	}()

	//手入力終了待ち受け
	go func() {
		stdin := bufio.NewScanner(os.Stdin)
		for {
			stdin.Scan()
			cmd := stdin.Text()
			if cmd == "quit" {
				done <- nil
			} else if cmd != "" {
				log.Printf("[%s] is not command\n", cmd)
			} else {
				ci <- true
			}
		}
	}()

	go func() {
		for <-ci {
			if lock() {
				err := RunTest()
				if err != nil {
					done <- err
				}
				unlock()
			}
		}
	}()

	//first run
	ci <- true

	return <-done
}

//Ignore
func Ignore(f string) bool {

	//".go"
	if len(f) < 3 {
		return true
	}
	if f[len(f)-3:] != ".go" {
		return true
	}

	//exist
	fi, err := os.Stat(f)
	if err != nil {
		return true
	}
	if fi.IsDir() {
		return true
	}
	return false
}

func RunTest() error {

	log.Println("\x1b[36m######################################################\x1b[0m")
	run := []string{"\x1b[36m$ go"}
	run = append(run, args...)
	run = append(run, "\x1b[0m")
	log.Println("Run", run)

	cmd := exec.Command("go", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return err
	}
	cmd.Start()

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		txt := scanner.Text()
		color := 37
		if strings.Index(txt, "FAIL") != -1 {
			color = 31
		} else if strings.Index(txt, "PASS") != -1 ||
			strings.Index(txt, "ok") != -1 {
			color = 32
		} else if strings.Index(txt, "RUN") != -1 {
			color = 35
		}
		log.Printf("\x1b[%dm%s\x1b[0m\n", color, txt)
	}
	err = cmd.Wait()

	if err != nil {
		log.Println("\x1b[31m" + err.Error() + "\x1b[0m")
	}

	log.Println("\x1b[36m# Press enter to execute.[quit] -> terminate.\x1b[0m")
	log.Println("\x1b[36m######################################################\x1b[0m")

	return nil
}
