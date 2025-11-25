package identity

import (
	"sync"

	"github.com/rs/xid"
)

type Identity struct {
	serviceName string
	instanceID  string
}

var (
	identity = Identity{
		serviceName: "unknown",
		instanceID:  xid.New().String(),
	}
	setServiceNameOnce sync.Once
)

// WhoAmI returns the global identity information
// serviceName can be set once during runtime. Before being set, it defaults to "unknown"
// instanceID is a unique identifier representing this execution of code. It is set at runtime initialization, and cannot be altered
func WhoAmI() (serviceName, instanceID string) {
	return identity.serviceName, identity.instanceID
}

// SetServiceName alters the global identity to use the provide service name
// This is protected by sync.Once so that the service name cannot be changed once set
// Do not set the service name in tests - rely on the default value if needed.
func SetServiceName(name string) {
	setServiceNameOnce.Do(func() {
		identity.serviceName = name
	})
}
