package s3

import "context"
import "errors"
import "flag"
import "log"

import "github.com/remram44/lock-run-cmd"
import "github.com/remram44/lock-run-cmd/internal/cli"

type S3LockingSystem struct{}

func Parse(args []string) (lockrun.LockingSystem, []string, error) {
	// Set up command line parser
	flagset := flag.NewFlagSet("k8s", flag.ExitOnError)
	cli.RegisterFlags(flagset)

	bucket := flagset.String("bucket", "", "Bucket to write to")

	object := flagset.String("object", "lock", "Object to write")

	if err := flagset.Parse(args); err != nil {
		return nil, nil, err
	}

	identity := cli.Identity()
	log.Printf("Using identity %v", identity)

	// Create S3 client
	// TODO
	log.Println(bucket, object)

	locking_system := S3LockingSystem{}
	return &locking_system, flagset.Args(), nil
}

func (ls *S3LockingSystem) Run(
	ctx context.Context,
	onLockAcquired func(),
	onLockLost func(),
) error {
	return errors.New("Unimplemented") // TODO
}

func (ls *S3LockingSystem) Stop() {
	// TODO
}
