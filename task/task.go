// Package task provides wrappers for simplified management of async functions.
package task

import "context"

// Task represents a background service.
type Task interface {
	// Run must execute the work of this service and block until
	// the context is cancelled, or until
	// the service is unable to continue due to an error.
	Run(context.Context) error

	// Name provides a human-friendly name for use in logging.
	Name() string
}
