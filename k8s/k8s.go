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
import electionResource "k8s.io/client-go/tools/leaderelection/resourcelock"

import "github.com/remram44/lock-run-cmd"
import "github.com/remram44/lock-run-cmd/internal/cli"

type K8sLockingSystem struct {
	namespace  string
	objectName string
	identity   string
	clientset  *kubernetes.Clientset
	ctxCancel  func()
}

func Parse(args []string) (lockrun.LockingSystem, []string, error) {
	// Set up command line parser
	flagset := flag.NewFlagSet("k8s", flag.ExitOnError)
	cli.RegisterFlags(flagset)

	kubeconfig := flagset.String("kubeconfig", "~/.kube/config", "Configuration file")

	inCluster := false
	flagset.BoolFunc("in-cluster", "Use in-cluster config", cli.SetBool(&inCluster))

	namespace := flagset.String("namespace", "default", "Kubernetes namespace")

	objectName := flagset.String("lease-object", "lock", "Lease object name")

	if err := flagset.Parse(args); err != nil {
		return nil, nil, err
	}

	identity := cli.Identity()
	log.Printf("Using identity %v", identity)

	kubeconfigArg := ""
	if !inCluster {
		kubeconfigArg = *kubeconfig
	}
	lockingSystem, err := New(
		kubeconfigArg,
		*namespace,
		*objectName,
		identity,
	)
	if err != nil {
		return nil, nil, err
	}

	return lockingSystem, flagset.Args(), nil
}

func New(kubeconfig string, namespace string, objectName string, identity string) (lockrun.LockingSystem, error) {
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

	lockingSystem := K8sLockingSystem{
		namespace:  namespace,
		objectName: objectName,
		clientset:  clientset,
		identity:   identity,
		ctxCancel:  nil,
	}
	return &lockingSystem, nil
}

func (ls *K8sLockingSystem) Run(
	ctx context.Context,
	onLockAcquired func(),
	onLockLost func(),
) error {
	// Kick off leaderelection code
	electCtx, electCancel := context.WithCancel(ctx)
	ls.ctxCancel = electCancel
	defer electCancel()
	lock := &electionResource.LeaseLock{
		LeaseMeta: k8smetav1.ObjectMeta{
			Name:      ls.objectName,
			Namespace: ls.namespace,
		},
		Client: ls.clientset.CoordinationV1(),
		LockConfig: electionResource.ResourceLockConfig{
			Identity: ls.identity,
		},
	}
	log.Print("election.RunOrDie()...")
	election.RunOrDie(electCtx, election.LeaderElectionConfig{
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
	ls.ctxCancel()
}

func (ls *K8sLockingSystem) Close() {
}
