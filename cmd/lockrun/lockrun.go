package main

import "context"
import "errors"
import "fmt"
import "log"
import "os"

import "github.com/remram44/lock-run-cmd"
import "github.com/remram44/lock-run-cmd/k8s"

func main() {
	// Get locking system from command line
	usage := func() {
		fmt.Printf("Usage: %v k8s ...\n", os.Args[0])
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error = nil
	var lockingSystem lockrun.LockingSystem = nil
	var args []string = nil
	switch os.Args[1] {
	case "help", "-help", "--help":
		usage()
		os.Exit(0)
	case "k8s":
		lockingSystem, args, err = k8s.Parse(os.Args[2:])
	case "etcd":
		err = errors.New("Not yet implemented")
	case "s3":
		err = errors.New("Not yet implemented")
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer lockingSystem.Close()

	// Create command
	cmd := lockrun.NewCommandRunner(args)

	// Run locking system
	err = lockingSystem.Run(
		context.Background(),
		func() {
			if err := cmd.Run(lockingSystem.Stop); err != nil {
				log.Fatal(err)
			}
		},
		func() {
			cmd.Stop()
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}
