package main

import "fmt"
import "log"
import "os"

import "github.com/remram44/lock-run-cmd/k8s"
import "github.com/remram44/lock-run-cmd/etcd"

func main() {
	usage := func() {
		fmt.Printf("Usage: %v [k8s|etcd] ...\n", os.Args[0])
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error = nil
	switch os.Args[1] {
	case "help", "-help", "--help":
		usage()
		os.Exit(0)
	case "k8s":
		err = k8s.Main(os.Args[2:])
	case "etcd":
		err = etcd.Main(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Fatal(err)
	}
}
