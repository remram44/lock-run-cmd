package etcd

import "context"
import "crypto/tls"
import "errors"
import "flag"
import "fmt"
import "log"
import "strings"
import "time"

import etcdv3 "go.etcd.io/etcd/client/v3"
import etcdtransport "go.etcd.io/etcd/client/pkg/v3/transport"

import "github.com/remram44/lock-run-cmd"
import "github.com/remram44/lock-run-cmd/internal/cli"

type EtcdLockingSystem struct {
	client   *etcdv3.Client
	identity string
	key      string
}

func Parse(args []string) (lockrun.LockingSystem, []string, error) {
	// Set up command line parser
	flagset := flag.NewFlagSet("k8s", flag.ExitOnError)
	cli.RegisterFlags(flagset)

	endpoints := flagset.String("endpoints", "127.0.0.1:2379", "Comma-separated list of host:port")

	ca_cert := flagset.String("cacert", "", "Path to CA certificate")

	client_cert := flagset.String("cert", "", "Path to client certificate")

	client_key := flagset.String("key", "", "Path to client key")

	username := flagset.String("username", "", "User name for authentication")

	password := flagset.String("password", "", "Password for authentication")

	key := flagset.String("lease-key", "lock", "ETCD key")

	if err := flagset.Parse(args); err != nil {
		return nil, nil, err
	}

	identity := cli.Identity()
	log.Printf("Using identity %v", identity)

	// Prepare TLS configuration if requested
	var tls_config *tls.Config = nil
	if *ca_cert != "" || *client_cert != "" || *client_key != "" {
		tls_info := etcdtransport.TLSInfo{
			TrustedCAFile: *ca_cert,
			CertFile:      *client_cert,
			KeyFile:       *client_key,
		}
		cfg, err := tls_info.ClientConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("Can't load certificates: %w", err)
		}
		tls_config = cfg
	}

	endpoints_slice := strings.Split(*endpoints, ",")
	locking_system, err := New(endpoints_slice,
		tls_config,
		*username,
		*password,
		*key,
		identity,
	)
	if err != nil {
		return nil, nil, err
	}

	return locking_system, flagset.Args(), nil
}

func New(
	endpoints []string,
	tls_config *tls.Config,
	username string,
	password string,
	key string,
	identity string,
) (lockrun.LockingSystem, error) {
	// Create ETCD client
	config := etcdv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tls_config,
		Username:    username,
		Password:    password,
	}
	client, err := etcdv3.New(config)
	if err != nil {
		return nil, err
	}

	locking_system := EtcdLockingSystem{
		client: client,
		key:    key,
	}
	return &locking_system, nil
}

func (ls *EtcdLockingSystem) Run(
	ctx context.Context,
	onLockAcquired func(),
	onLockLost func(),
) error {
	// Get a lease
	resp, err := ls.client.Grant(ctx, int64(cli.LeaseDuration().Seconds()))
	if err != nil {
		return fmt.Errorf("Can't get etcd lease: %w", err)
	}
	lease := resp.ID

	// Read key
	txn, err := ls.client.Txn(ctx).
		// If the key doesn't exist
		If(etcdv3.Compare(etcdv3.CreateRevision(ls.key), "=", 0)).
		// Create it with our identity
		Then(etcdv3.OpPut(ls.key, ls.identity, etcdv3.WithLease(lease))).
		// Else read TTL
		Else(etcdv3.OpGet(ls.key, etcdv3.WithCountOnly())).
		Commit()
	if err != nil {
		return fmt.Errorf("Can't execute etcd transaction: %w", err)
	}
	if txn.Succeeded {
		// TODO: We are locked
	} else {
		txn.Responses[0].GetResponseRange()
	}

	return errors.New("Unimplemented") // TODO
}

func (ls *EtcdLockingSystem) Stop() {
	// TODO
}

func (ls *EtcdLockingSystem) Close() {
	ls.client.Close()
}
