package common

import "crypto/rand"
import "encoding/hex"
import "flag"
import "fmt"
import "log"
import "os"
import "os/signal"
import "syscall"
import "time"

var leaseInterval = flag.Duration(
	"lease-interval",
	time.Duration(15*time.Second),
	"Interval between lease renewal",
)
var leaseDuration *time.Duration = nil
var gracePeriod = flag.Duration(
	"grace-period",
	time.Duration(5*time.Second),
	"Grace period between SIGTERM and SIGKILL",
)
var identity *string = nil

func RegisterFlags(fs *flag.FlagSet) {
	fs.Func("lease-duration", "Length of the lease", func(arg string) error {
		var duration time.Duration
		var err error
		if duration, err = time.ParseDuration(arg); err != nil {
			return err
		}
		leaseDuration = &duration
		return nil
	})

	fs.Func("identity", "Identity of this process", func(arg string) error {
		identity = &arg
		return nil
	})
}

func LeaseInterval() time.Duration {
	return *leaseInterval
}

func LeaseDuration() time.Duration {
	if leaseDuration != nil {
		return *leaseDuration
	} else {
		return *leaseInterval * 2
	}
}

func GracePeriod() time.Duration {
	return *gracePeriod
}

func Identity() string {
	if identity == nil {
		new_identity := RandomIdentity()
		identity = &new_identity
	}
	return *identity
}

func SetBool(target *bool) func(string) error {
	return func(arg string) error {
		switch arg {
		case "true":
			*target = true
		case "false":
			*target = false
		default:
			return fmt.Errorf("invalid boolean: %v", arg)
		}
		return nil
	}
}

func RandomIdentity() string {
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

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
