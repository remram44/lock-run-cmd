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

	caCert := flagset.String("cacert", "", "Path to CA certificate")

	clientCert := flagset.String("cert", "", "Path to client certificate")

	clientKey := flagset.String("key", "", "Path to client key")

	username := flagset.String("username", "", "User name for authentication")

	password := flagset.String("password", "", "Password for authentication")

	key := flagset.String("lease-key", "lock", "ETCD key")

	if err := flagset.Parse(args); err != nil {
		return nil, nil, err
	}

	identity := cli.Identity()
	log.Printf("Using identity %v", identity)

	// Prepare TLS configuration if requested
	var tlsConfig *tls.Config = nil
	if *caCert != "" || *clientCert != "" || *clientKey != "" {
		tlsInfo := etcdtransport.TLSInfo{
			TrustedCAFile: *caCert,
			CertFile:      *clientCert,
			KeyFile:       *clientKey,
		}
		cfg, err := tlsInfo.ClientConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("Can't load certificates: %w", err)
		}
		tlsConfig = cfg
	}

	endpointsSlice := strings.Split(*endpoints, ",")
	lockingSystem, err := New(endpointsSlice,
		tlsConfig,
		*username,
		*password,
		*key,
		identity,
	)
	if err != nil {
		return nil, nil, err
	}

	return lockingSystem, flagset.Args(), nil
}

func New(
	endpoints []string,
	tlsConfig *tls.Config,
	username string,
	password string,
	key string,
	identity string,
) (lockrun.LockingSystem, error) {
	// Create ETCD client
	config := etcdv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
		Username:    username,
		Password:    password,
	}
	client, err := etcdv3.New(config)
	if err != nil {
		return nil, err
	}

	lockingSystem := EtcdLockingSystem{
		client: client,
		key:    key,
	}
	return &lockingSystem, nil
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
