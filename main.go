package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/xerrors"
)

var (
	Cmd       string
	SubCmd    string
	CmdArgs   string
	ArgsSlice []string
	WatchPath string
	Duration  int64
)

func init() {
	flag.StringVar(&Cmd, "c", "go", "Command Name")
	flag.StringVar(&SubCmd, "s", "test", "Sub Command Name")
	flag.StringVar(&CmdArgs, "a", "-v ./...", "Command Arguments")
	flag.Int64Var(&Duration, "d", 5, "Lock Duration")
	flag.StringVar(&WatchPath, "w", "./", "Watch path")
}

func main() {

	flag.Parse()

	err := circuit(os.Stdout, os.Stdin)
	if err != nil {
		fmt.Printf("error %+v", err)
		os.Exit(1)
	}

	fmt.Println("bye!")
	os.Exit(0)
}

var lastUnixTime int64

func lock() bool {
	now := time.Now().Unix()
	if now > lastUnixTime+Duration {
		return true
	}

	log.Println("Please wait for a while and then execute.")
	return false
}

func unlock() {
	now := time.Now()
	lastUnixTime = now.Unix()
}

func circuit(w io.Writer, r io.Reader) error {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return xerrors.Errorf("fsnotify.NewWatcher(): %w", err)
	}
	defer watcher.Close()

	err = watcher.Add(WatchPath)
	if err != nil {
		return xerrors.Errorf("Watcher.Add(%s): %w", WatchPath, err)
	}

	args := strings.Split(CmdArgs, " ")
	ArgsSlice = make([]string, 1+len(args))
	ArgsSlice[0] = SubCmd
	for idx, arg := range args {
		ArgsSlice[idx+1] = arg
	}

	done := make(chan error)
	ci := make(chan bool)

	defer close(ci)
	defer close(done)

	go func() {
		for {
			select {
			case e := <-watcher.Events:

				if !Ignore(e) {
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
func Ignore(e fsnotify.Event) bool {

	switch {
	case e.Op&fsnotify.Rename == fsnotify.Rename:
	case e.Op&fsnotify.Create == fsnotify.Create:
	case e.Op&fsnotify.Remove == fsnotify.Remove:
	case e.Op&fsnotify.Chmod == fsnotify.Chmod:
	case e.Op&fsnotify.Write == fsnotify.Write:
	default:
	}

	f := e.Name

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

	cmd := exec.Command(Cmd, ArgsSlice...)
	log.Println("Run", cmd.Args)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return xerrors.Errorf("command StdoutPipe(): %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return xerrors.Errorf("command StderrPipe(): %w", err)
	}

	err = cmd.Start()
	if err != nil {
		printStdError(stderr)
		return xerrors.Errorf("command Start(): %w", err)
	}

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
		printStdError(stderr)
	}

	log.Println("\x1b[36m# 'quit' -> terminate.\x1b[0m")
	log.Println("\x1b[36m# Press enter to execute.\x1b[0m")
	log.Println("\x1b[36m######################################################\x1b[0m")

	return nil
}

func printStdError(r io.Reader) {
	std, err := ioutil.ReadAll(r)
	if err != nil {
		if !xerrors.Is(err, os.ErrClosed) {
			//log.Printf("%s\n", err)
		}
		return
	}
	log.Printf("%s\n", std)
}
