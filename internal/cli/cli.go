package cli

import "crypto/rand"
import "encoding/hex"
import "flag"
import "fmt"
import "log"
import "time"

var leaseInterval = flag.Duration(
	"lease-interval",
	time.Duration(15*time.Second),
	"Interval between lease renewal",
)
var leaseDuration *time.Duration = nil
var gracePeriod = flag.Duration(
	"grace-period",
	time.Duration(5*time.Second),
	"Grace period between SIGTERM and SIGKILL",
)
var identity *string = nil

func RegisterFlags(fs *flag.FlagSet) {
	fs.Func("lease-duration", "Length of the lease", func(arg string) error {
		var duration time.Duration
		var err error
		if duration, err = time.ParseDuration(arg); err != nil {
			return err
		}
		leaseDuration = &duration
		return nil
	})

	fs.Func("identity", "Identity of this process", func(arg string) error {
		identity = &arg
		return nil
	})
}

func LeaseInterval() time.Duration {
	return *leaseInterval
}

func LeaseDuration() time.Duration {
	if leaseDuration != nil {
		return *leaseDuration
	} else {
		return *leaseInterval * 2
	}
}

func GracePeriod() time.Duration {
	return *gracePeriod
}

func RandomIdentity() string {
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

func Identity() string {
	if identity == nil {
		newIdentity := RandomIdentity()
		identity = &newIdentity
	}
	return *identity
}

func SetBool(target *bool) func(string) error {
	return func(arg string) error {
		switch arg {
		case "true":
			*target = true
		case "false":
			*target = false
		default:
			return fmt.Errorf("invalid boolean: %v", arg)
		}
		return nil
	}
}
