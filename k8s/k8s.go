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

	identity, err := common.RandomIdentity()
	if err != nil {
		return err
	}
	log.Printf("Using identity %v", identity)

	// Debug
	log.Printf("kubeconfig=%v in_cluster=%v namespace=%v lease-interval=%v lease-duration=%v", *kubeconfig, in_cluster, namespace, common.LeaseInterval(), common.LeaseDuration())

	// Create Kubernetes API client
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

	// Create a Context, cancelled on SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		log.Print("Received SIGTERM, shutting down")
		cancel()
	}()

	// Kick off leaderelection code
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
	election.RunOrDie(ctx, election.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   common.LeaseDuration(),
		RenewDeadline:   common.LeaseInterval(),
		RetryPeriod:     5 * time.Second,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// TODO
			},
			OnStoppedLeading: func() {
				// TODO
			},
		},
	})

	return nil
}
