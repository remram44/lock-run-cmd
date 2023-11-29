package etcd

import "context"
import "errors"
import "flag"
import "log"

import "github.com/remram44/lock-run-cmd"
import "github.com/remram44/lock-run-cmd/internal/cli"

type EtcdLockingSystem struct{}

func Parse(args []string) (lockrun.LockingSystem, []string, error) {
	// Set up command line parser
	flagset := flag.NewFlagSet("k8s", flag.ExitOnError)
	cli.RegisterFlags(flagset)

	endpoints := flagset.String("endpoints", "127.0.0.1:2379", "Comma-separated list of host:port")

	ca_cert := flagset.String("cacert", "", "Path to CA certificate")

	client_cert := flagset.String("cert", "", "Path to client certificate")

	client_key := flagset.String("key", "", "Path to client key")

	key := flagset.String("lease-key", "lock", "ETCD key")

	if err := flagset.Parse(args); err != nil {
		return nil, nil, err
	}

	identity := cli.Identity()
	log.Printf("Using identity %v", identity)

	// Create ETCD client
	// TODO
	log.Println(endpoints, ca_cert, client_cert, client_key, key)

	locking_system := EtcdLockingSystem{}
	return &locking_system, flagset.Args(), nil
}

func (ls *EtcdLockingSystem) Run(
	ctx context.Context,
	onLockAcquired func(),
	onLockLost func(),
) error {
	return errors.New("Unimplemented") // TODO
}

func (ls *EtcdLockingSystem) Stop() {
	// TODO
}

func (ls *EtcdLockingSystem) Close() {
}
