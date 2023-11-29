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

type K8sLockingSystem struct {
	namespace   string
	object_name string
	identity    string
	clientset   *kubernetes.Clientset
	ctx_cancel  func()
}

func Parse(args []string) (lockrun.LockingSystem, []string, error) {
	// Set up command line parser
	flagset := flag.NewFlagSet("k8s", flag.ExitOnError)
	cli.RegisterFlags(flagset)

	kubeconfig := flagset.String("kubeconfig", "~/.kube/config", "Configuration file")

	in_cluster := false
	flagset.BoolFunc("in-cluster", "Use in-cluster config", cli.SetBool(&in_cluster))

	namespace := flagset.String("namespace", "default", "Kubernetes namespace")

	object_name := flagset.String("lease-object", "lock", "Lease object name")

	if err := flagset.Parse(args); err != nil {
		return nil, nil, err
	}

	identity := cli.Identity()
	log.Printf("Using identity %v", identity)

	kubeconfig_arg := ""
	if !in_cluster {
		kubeconfig_arg = *kubeconfig
	}
	locking_system, err := New(
		kubeconfig_arg,
		*namespace,
		*object_name,
		identity,
	)
	if err != nil {
		return nil, nil, err
	}

	return locking_system, flagset.Args(), nil
}

func New(kubeconfig string, namespace string, object_name string, identity string) (lockrun.LockingSystem, error) {
	// Create Kubernetes API client
	var config *k8srest.Config
	var err error
	if kubeconfig == "" {
		config, err = k8srest.InClusterConfig()
	} else {
		config, err = k8sclientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("Can't load Kubernetes config: %w", err)
	}
	config.UserAgent = "lock-run-cmd"
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	if identity == "" {
		identity = cli.RandomIdentity()
	}

	locking_system := K8sLockingSystem{
		namespace:   namespace,
		object_name: object_name,
		clientset:   clientset,
		identity:    identity,
		ctx_cancel:  nil,
	}
	return &locking_system, nil
}

func (ls *K8sLockingSystem) Run(
	ctx context.Context,
	onLockAcquired func(),
	onLockLost func(),
) error {
	// Kick off leaderelection code
	elect_ctx, elect_cancel := context.WithCancel(ctx)
	ls.ctx_cancel = elect_cancel
	defer elect_cancel()
	lock := &election_resource.LeaseLock{
		LeaseMeta: k8smetav1.ObjectMeta{
			Name:      ls.object_name,
			Namespace: ls.namespace,
		},
		Client: ls.clientset.CoordinationV1(),
		LockConfig: election_resource.ResourceLockConfig{
			Identity: ls.identity,
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
				onLockAcquired()
			},
			OnStoppedLeading: func() {
				log.Print("OnStoppedLeading()")
				onLockLost()
			},
		},
	})

	return nil
}

func (ls *K8sLockingSystem) Stop() {
	ls.ctx_cancel()
}

func (ls *K8sLockingSystem) Close() {
}
