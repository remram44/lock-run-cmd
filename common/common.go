package common

import "errors"
import "flag"
import "fmt"
import "time"

var LeaseInterval = flag.Duration(
	"lease-interval",
	time.Duration(15*time.Second),
	"Interval between lease renewal",
)
var LeaseDuration *time.Duration = nil

func RegisterFlags(fs *flag.FlagSet) {
	fs.Func("lease-duration", "Length of the lease", func(arg string) error {
		var duration time.Duration
		var err error
		if duration, err = time.ParseDuration(arg); err != nil {
			return err
		}
		LeaseDuration = &duration
		return nil
	})
}

func SetBool(target *bool) func(string) error {
	return func(arg string) error {
		switch arg {
		case "true":
			*target = true
		case "false":
			*target = false
		default:
			return errors.New(fmt.Sprintf("invalid boolean: %v", arg))
		}
		return nil
	}
}
