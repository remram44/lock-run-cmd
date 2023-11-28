package k8s

import "context"
import "flag"
import "fmt"
import "log"
import "os"
import "os/signal"
import "syscall"
import "time"

import k8sclientcmd "k8s.io/client-go/tools/clientcmd"
import k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
import k8srest "k8s.io/client-go/rest"
import "k8s.io/client-go/kubernetes"
import election "k8s.io/client-go/tools/leaderelection"
import election_resource "k8s.io/client-go/tools/leaderelection/resourcelock"

import "github.com/remram44/lock-run-cmd/common"

func Main(args []string) error {
	// Set up command line parser
	cli := flag.NewFlagSet("k8s", flag.ExitOnError)
	common.RegisterFlags(cli)

	kubeconfig := cli.String("kubeconfig", "~/.kube/config", "Configuration file")

	in_cluster := false
	cli.BoolFunc("in-cluster", "Use in-cluster config", common.SetBool(&in_cluster))

	namespace := cli.String("namespace", "default", "Kubernetes namespace")

	object_name := cli.String("lease-object", "lock", "Lease object name")

	if err := cli.Parse(args); err != nil {
		return err
	}

	identity := common.Identity()
	log.Printf("Using identity %v", identity)

	// Debug
	log.Printf("kubeconfig=%v in_cluster=%v namespace=%v lease-interval=%v lease-duration=%v", *kubeconfig, in_cluster, namespace, common.LeaseInterval(), common.LeaseDuration())

	// Create Kubernetes API client
	var err error
	var config *k8srest.Config
	if in_cluster {
		config, err = k8srest.InClusterConfig()
	} else {
		config, err = k8sclientcmd.BuildConfigFromFlags("", *kubeconfig)
	}
	if err != nil {
		return fmt.Errorf("Can't load Kubernetes config: %w", err)
	}
	config.UserAgent = "lock-run-cmd"
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Create command
	cmd_args := []string{"/bin/sh", "-c", "while true; do sleep 5; echo running; done"}
	var process *os.Process = nil
	process_exited := make(chan *os.ProcessState, 1)

	// Forward SIGTERM
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			<-ch
			log.Print("Received SIGTERM, forwarding")
			if process != nil {
				process.Signal(syscall.SIGINT)
			}
		}
	}()

	// Kick off leaderelection code
	elect_ctx, elect_cancel := context.WithCancel(context.Background())
	defer elect_cancel()
	lock := &election_resource.LeaseLock{
		LeaseMeta: k8smetav1.ObjectMeta{
			Name:      *object_name,
			Namespace: *namespace,
		},
		Client: clientset.CoordinationV1(),
		LockConfig: election_resource.ResourceLockConfig{
			Identity: identity,
		},
	}
	log.Print("election.RunOrDie()...")
	election.RunOrDie(elect_ctx, election.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   common.LeaseDuration(),
		RenewDeadline:   common.LeaseInterval(),
		RetryPeriod:     5 * time.Second,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Print("OnStartedLeading()")
				log.Print("StartProcess()...")
				process, err = os.StartProcess(
					cmd_args[0],
					cmd_args,
					&os.ProcAttr{
						// Inherit pipes
						Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
					},
				)
				if err != nil {
					log.Printf("Error running command: %v", err)
					log.Print("elect_cancel()")
					elect_cancel()
				} else {
					log.Print("cmd started")
				}

				// Start a goroutine to send the process exit state on a channel
				go func() {
					state, err := process.Wait()
					if err != nil {
						log.Fatalf("Error waiting for command: %v", err)
					}
					log.Print("command exited")
					process_exited <- state
					log.Print("elect_cancel()")
					elect_cancel()
				}()
			},
			OnStoppedLeading: func() {
				log.Print("OnStoppedLeading()")
				log.Print("Sending SIGTERM...")
				process.Signal(syscall.SIGINT)
				select {
				case state := <-process_exited:
					if state.Exited() {
						log.Printf("Process exited with status %v", state.ExitCode())
					} else {
						log.Printf("Process was terminated by a signal")
					}
				case <-time.After(common.GracePeriod()):
					log.Print("Grace period elapsed, sending SIGKILL...")
					process.Signal(syscall.SIGKILL)
					log.Print("elect_cancel()")
					elect_cancel()
				}
			},
		},
	})

	return nil
}
