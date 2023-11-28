package lockrun

import "context"

type LockingSystem interface {
	Run(
		ctx context.Context,
		onLockAcquired func(),
		onLockLost func(),
	) error
	Stop()
}
