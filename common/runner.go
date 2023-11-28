package common

import "fmt"
import "log"
import "os"
import "os/signal"
import "syscall"
import "time"

type CommandRunner struct {
	args    []string
	process *os.Process
	exited  chan *os.ProcessState
}

func NewCommandRunner(args []string) *CommandRunner {
	runner := CommandRunner{
		args:    args,
		process: nil,
		exited:  make(chan *os.ProcessState, 1),
	}
	return &runner
}

func (runner *CommandRunner) Run(cancel func()) error {
	log.Print("StartProcess()...")
	var err error
	runner.process, err = os.StartProcess(
		runner.args[0],
		runner.args,
		&os.ProcAttr{
			// Inherit pipes
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		},
	)
	if err != nil {
		return fmt.Errorf("Error running command: %w", err)
	} else {
		log.Print("cmd started")
	}

	// Forward SIGTERM
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			<-ch
			log.Print("Received SIGTERM, forwarding")
			runner.process.Signal(syscall.SIGINT)
		}
	}()

	// Start a goroutine to send the process exit state on a channel
	go func() {
		state, err := runner.process.Wait()
		if err != nil {
			log.Fatalf("Error waiting for command: %v", err)
		}
		log.Print("command exited")
		runner.exited <- state
		log.Print("elect_cancel()")
		cancel()
	}()

	return nil
}

func (runner *CommandRunner) Stop() {
	if runner.process == nil {
		return
	}
	log.Print("Sending SIGTERM...")
	runner.process.Signal(syscall.SIGINT)
	select {
	case state := <-runner.exited:
		if state.Exited() {
			log.Printf("Process exited with status %v", state.ExitCode())
		} else {
			log.Printf("Process was terminated by a signal")
		}
	case <-time.After(GracePeriod()):
		log.Print("Grace period elapsed, sending SIGKILL...")
		runner.process.Signal(syscall.SIGKILL)
	}
}
