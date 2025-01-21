package healthcheck

type (
	HealthCheck struct {
		PingTime string `json:"pingTime"`
	}
)

func NewHealthCheck(pingTime string) *HealthCheck {
	return &HealthCheck{PingTime: pingTime}
}
