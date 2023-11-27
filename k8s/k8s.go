package k8s

import "context"
import "flag"
import "fmt"
import "log"

import k8sclientcmd "k8s.io/client-go/tools/clientcmd"
import k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
import k8srest "k8s.io/client-go/rest"
import "k8s.io/client-go/kubernetes"

import "github.com/remram44/lock-run-cmd/common"

func Main(args []string) error {
	// Set up command line parser
	cli := flag.NewFlagSet("k8s", flag.ExitOnError)
	common.RegisterFlags(cli)

	kubeconfig := cli.String("kubeconfig", "~/.kube/config", "Configuration file")

	in_cluster := false
	cli.BoolFunc("in-cluster", "Use in-cluster config", common.SetBool(&in_cluster))

	namespace := cli.String("namespace", "default", "Kubernetes namespace")

	if err := cli.Parse(args); err != nil {
		return err
	}

	// Debug
	log.Printf("kubeconfig=%v in_cluster=%v namespace=%v lease-interval=%v lease-duration=%v", *kubeconfig, in_cluster, namespace, common.LeaseInterval, common.LeaseDuration)

	// Create Kubernetes API client
	var config *k8srest.Config
	var err error
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
	configMapClient := clientset.CoreV1().ConfigMaps(*namespace)

	// Read a ConfigMap
	cm, err := configMapClient.Get(context.TODO(), "test", k8smetav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Print(cm)

	return nil
}
