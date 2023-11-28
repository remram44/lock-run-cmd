package k8s

import "context"
import "flag"
import "fmt"
import "log"
import "time"

import k8sclientcmd "k8s.io/client-go/tools/clientcmd"
import k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
import k8srest "k8s.io/client-go/rest"
import "k8s.io/client-go/kubernetes"
import election "k8s.io/client-go/tools/leaderelection"
import election_resource "k8s.io/client-go/tools/leaderelection/resourcelock"

import "github.com/remram44/lock-run-cmd"
import "github.com/remram44/lock-run-cmd/internal/cli"

func Main(args []string) error {
	// Set up command line parser
	flagset := flag.NewFlagSet("k8s", flag.ExitOnError)
	cli.RegisterFlags(flagset)

	kubeconfig := flagset.String("kubeconfig", "~/.kube/config", "Configuration file")

	in_cluster := false
	flagset.BoolFunc("in-cluster", "Use in-cluster config", cli.SetBool(&in_cluster))

	namespace := flagset.String("namespace", "default", "Kubernetes namespace")

	object_name := flagset.String("lease-object", "lock", "Lease object name")

	if err := flagset.Parse(args); err != nil {
		return err
	}

	identity := cli.Identity()
	log.Printf("Using identity %v", identity)

	// Debug
	log.Printf("kubeconfig=%v in_cluster=%v namespace=%v lease-interval=%v lease-duration=%v", *kubeconfig, in_cluster, namespace, cli.LeaseInterval(), cli.LeaseDuration())

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
	cmd := lockrun.NewCommandRunner(flagset.Args())

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
		LeaseDuration:   cli.LeaseDuration(),
		RenewDeadline:   cli.LeaseInterval(),
		RetryPeriod:     5 * time.Second,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Print("OnStartedLeading()")
				if err := cmd.Run(elect_cancel); err != nil {
					log.Fatal(err)
				}
			},
			OnStoppedLeading: func() {
				log.Print("OnStoppedLeading()")
				cmd.Stop()
				log.Print("elect_cancel()")
				elect_cancel()
			},
		},
	})

	return nil
}
