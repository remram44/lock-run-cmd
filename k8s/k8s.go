package k8s

import "flag"
import "log"

import "github.com/remram44/lock-run-cmd/common"

func Main(args []string) error {
	cli := flag.NewFlagSet("k8s", flag.ExitOnError)
	common.RegisterFlags(cli)

	kubeconfig := cli.String("kubeconfig", "~/.kube/config", "Configuration file")

	in_cluster := false
	cli.BoolFunc("in-cluster", "Use in-cluster config", common.SetBool(&in_cluster))

	var namespace *string = nil
	cli.Func("namespace", "Kubernetes namespace", func(arg string) error {
		namespace = &arg
		return nil
	})

	if err := cli.Parse(args); err != nil {
		return err
	}

	var show_namespace string
	if namespace == nil {
		show_namespace = "(unset)"
	} else {
		show_namespace = *namespace
	}
	log.Printf("kubeconfig=%v in_cluster=%v namespace=%v lease-interval=%v lease-duration=%v", *kubeconfig, in_cluster, show_namespace, common.LeaseInterval, common.LeaseDuration)

	return nil
}
