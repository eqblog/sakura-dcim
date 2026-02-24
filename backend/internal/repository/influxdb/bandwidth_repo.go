package influxdb

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

type BandwidthRepo struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	queryAPI api.QueryAPI
	bucket   string
	org      string
}

func NewBandwidthRepo(client influxdb2.Client, cfg *config.InfluxConfig) *BandwidthRepo {
	return &BandwidthRepo{
		client:   client,
		writeAPI: client.WriteAPIBlocking(cfg.Org, cfg.Bucket),
		queryAPI: client.QueryAPI(cfg.Org),
		bucket:   cfg.Bucket,
		org:      cfg.Org,
	}
}

func (r *BandwidthRepo) WriteBandwidthPoint(ctx context.Context, portID, switchID string, inBytes, outBytes uint64, inBps, outBps float64) error {
	p := influxdb2.NewPoint("bandwidth",
		map[string]string{
			"port_id":   portID,
			"switch_id": switchID,
		},
		map[string]interface{}{
			"in_bytes":  int64(inBytes),
			"out_bytes": int64(outBytes),
			"in_bps":    inBps,
			"out_bps":   outBps,
		},
		time.Now(),
	)
	return r.writeAPI.WritePoint(ctx, p)
}

func (r *BandwidthRepo) WriteSensorReading(ctx context.Context, serverID, sensorName, sensorType string, value float64, status string) error {
	p := influxdb2.NewPoint("sensor",
		map[string]string{
			"server_id":   serverID,
			"sensor_name": sensorName,
			"sensor_type": sensorType,
		},
		map[string]interface{}{
			"value":  value,
			"status": status,
		},
		time.Now(),
	)
	return r.writeAPI.WritePoint(ctx, p)
}

func (r *BandwidthRepo) QueryBandwidth(ctx context.Context, portID string, start, stop time.Time) ([]domain.BandwidthDataPoint, error) {
	query := fmt.Sprintf(`from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "bandwidth" and r.port_id == "%s")
		|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")`,
		r.bucket, start.Format(time.RFC3339), stop.Format(time.RFC3339), portID)

	result, err := r.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("influx query: %w", err)
	}

	var points []domain.BandwidthDataPoint
	for result.Next() {
		record := result.Record()
		dp := domain.BandwidthDataPoint{
			Timestamp: record.Time(),
		}
		if v, ok := record.ValueByKey("in_bytes").(int64); ok {
			dp.InBytes = uint64(v)
		}
		if v, ok := record.ValueByKey("out_bytes").(int64); ok {
			dp.OutBytes = uint64(v)
		}
		if v, ok := record.ValueByKey("in_bps").(float64); ok {
			dp.InBps = v
		}
		if v, ok := record.ValueByKey("out_bps").(float64); ok {
			dp.OutBps = v
		}
		points = append(points, dp)
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	return points, nil
}

func (r *BandwidthRepo) QuerySensors(ctx context.Context, serverID string, start, stop time.Time) ([]domain.SensorDataPoint, error) {
	query := fmt.Sprintf(`from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "sensor" and r.server_id == "%s")
		|> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")`,
		r.bucket, start.Format(time.RFC3339), stop.Format(time.RFC3339), serverID)

	result, err := r.queryAPI.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("influx query: %w", err)
	}

	var points []domain.SensorDataPoint
	for result.Next() {
		record := result.Record()
		dp := domain.SensorDataPoint{
			Timestamp:  record.Time(),
			SensorName: fmt.Sprintf("%v", record.ValueByKey("sensor_name")),
			SensorType: fmt.Sprintf("%v", record.ValueByKey("sensor_type")),
		}
		if v, ok := record.ValueByKey("value").(float64); ok {
			dp.Value = v
		}
		if v, ok := record.ValueByKey("status").(string); ok {
			dp.Status = v
		}
		points = append(points, dp)
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	return points, nil
}

func (r *BandwidthRepo) Close() {
	r.client.Close()
}
