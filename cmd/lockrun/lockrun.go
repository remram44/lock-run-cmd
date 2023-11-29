package main

import "context"
import "fmt"
import "log"
import "os"

import "github.com/remram44/lock-run-cmd"
import "github.com/remram44/lock-run-cmd/k8s"
import "github.com/remram44/lock-run-cmd/etcd"
import "github.com/remram44/lock-run-cmd/s3"

func main() {
	// Get locking system from command line
	usage := func() {
		fmt.Printf("Usage: %v [k8s|etcd] ...\n", os.Args[0])
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error = nil
	var locking_system lockrun.LockingSystem = nil
	var args []string = nil
	switch os.Args[1] {
	case "help", "-help", "--help":
		usage()
		os.Exit(0)
	case "k8s":
		locking_system, args, err = k8s.Parse(os.Args[2:])
	case "etcd":
		locking_system, args, err = etcd.Parse(os.Args[2:])
	case "s3":
		locking_system, args, err = s3.Parse(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer locking_system.Close()

	// Create command
	cmd := lockrun.NewCommandRunner(args)

	// Run locking system
	err = locking_system.Run(
		context.Background(),
		func() {
			if err := cmd.Run(locking_system.Stop); err != nil {
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
