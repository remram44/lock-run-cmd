package common

import "crypto/rand"
import "encoding/hex"
import "flag"
import "fmt"
import "time"

var leaseInterval = flag.Duration(
	"lease-interval",
	time.Duration(15*time.Second),
	"Interval between lease renewal",
)
var leaseDuration *time.Duration = nil

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

func RandomIdentity() (string, error) {
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
