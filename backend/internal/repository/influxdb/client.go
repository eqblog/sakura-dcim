package influxdb

import (
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
)

func NewClient(cfg *config.InfluxConfig) influxdb2.Client {
	return influxdb2.NewClient(cfg.URL, cfg.Token)
}
