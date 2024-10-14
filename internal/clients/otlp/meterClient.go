package otlp

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type IMeterClient interface {
	GetMeterProvider(ctx context.Context) (*metric.MeterProvider, error)
}

type MeterClient struct {
	collectorUrl string
	serviceName  string
}

func NewMeterClient(collectorUrl, serviceName string) *MeterClient {
	return &MeterClient{
		collectorUrl: collectorUrl,
		serviceName:  serviceName,
	}
}

func (m *MeterClient) GetMeterProvider(ctx context.Context) (*metric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(m.collectorUrl),
	)
	if err != nil {
		log.Fatalf("failed to create the OTLP exporter: %v", err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(resource.NewWithAttributes(
			"service.name", attribute.String("service.name", m.serviceName),
		)),
	)
	otel.SetMeterProvider(meterProvider)

	return meterProvider, nil
}
