package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

var verbose = flag.Bool("verbose", false, "Print verbose output")
var delay = flag.Int("delay", 0, "A time in milliseconds to wait before starting the sub process")
var wait = flag.Int("wait", 0, "A time in milliseconds to wait between restarting the sub process on exit")

func main() {
	flag.Parse()

	subCommand := flag.Args()
	if len(subCommand) == 0 {
		fatalf("please provide a sub command to run")
	}

	pwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	if err := runLoop(quit, pwd, subCommand...); err != nil {
		fatal(err)
	}

	os.Exit(0)
}

func createSub(pwd string, subCommand ...string) (*exec.Cmd, error) {
	bin := subCommand[0]

	binPath, err := exec.LookPath(bin)
	if err != nil {
		return nil, err
	}

	sub := exec.Command(binPath, subCommand[1:]...)
	sub.Env = os.Environ()
	sub.Dir = pwd
	sub.Stdout = os.Stdout
	sub.Stderr = os.Stderr

	return sub, nil
}

func runLoop(quit chan os.Signal, pwd string, subCommand ...string) error {
	if delay != nil && *delay > 0 {
		delayMillis := time.Duration(*delay) * time.Millisecond
		verbosef("delaying %v before starting", delayMillis)
		alarm := time.After(delayMillis)
		select {
		case <-alarm:
			break
		case <-quit:
			verbosef("received SIGINT during delay, exiting")
			return nil
		}
	}

	var sub *exec.Cmd
	var err error
	var didQuit bool
	var abort chan struct{}
	var aborted chan struct{}

	for {
		abort = make(chan struct{})
		aborted = make(chan struct{})

		sub, err = createSub(pwd, subCommand...)
		if err != nil {
			return err
		}

		if err := sub.Start(); err != nil {
			return err
		}

		// kick off monitor
		go func() {
			select {
			case <-quit:
				verbosef("received SIGINT while sub process is running, killing sub process")
				didQuit = true
				sub.Process.Kill()
				return
			case <-abort:
				close(aborted)
				return
			}
		}()

		// wait for the sub process to exit
		if err := sub.Wait(); err != nil {
			verbosef("sub process exit: %v", err)
		}

		if didQuit {
			return nil
		}

		// abort the monitor
		close(abort)
		// wait for monitor to return
		<-aborted

		if wait != nil && *wait > 0 {
			waitMillis := time.Duration(*wait) * time.Millisecond
			verbosef("waiting %v before restart", waitMillis)
			alarm := time.After(waitMillis)
			select {
			case <-alarm:
				continue
			case <-quit:
				verbosef("received SIGINT during wait, exiting")
				return nil
			}
		}
	}
}

func verbosef(format string, args ...interface{}) {
	if verbose != nil && *verbose {
		fmt.Fprintf(os.Stdout, "recover: "+format+"\n", args...)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "recover: "+format+"\n", args...)
	os.Exit(1)
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "recover: %v\n", err)
		os.Exit(1)
	}
}
