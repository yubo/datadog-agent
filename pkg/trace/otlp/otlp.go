package otlp

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.uber.org/zap"
)

func zapWrapper() *zap.Logger {
	// TODO: Create zap wrapper around current Agent logger.
	// We can replace the core so that the Write method just calls the
	// pkg/util/log functions for logging.
	return zap.NewNop()
}

func getOTLPReceiverConfiguration() config.Receiver {
	// TODO This configuration should be built from configuration that we
	// get from pkg/config.
	// The configuration has the supported protocols and the endpoints for these.
	factory := otlpreceiver.NewFactory()
	return factory.CreateDefaultConfig()
}

// GetOTLPReceiver creates an OTLP traces format receiver
//  We would use this like the following
//
//    receiver, _ := GetOTLPReceiver(getOTLPReceiverConfiguration())
//    // Start consuming traces
//    receiver.Start(context.TODO(), <some reasonable component.Host value>)
//    // ....
//    receiver.Shutdown(context.TODO())
//
func GetOTLPReceiver(config config.Receiver) (component.TracesReceiver, error) {
	factory := otlpreceiver.NewFactory()
	consumer, err := newConsumer()
	if err != nil {
		return nil, err
	}
	return factory.CreateTracesReceiver(
		context.TODO(),
		component.ReceiverCreateParams{
			Logger:               zapWrapper(),
			ApplicationStartInfo: component.ApplicationStartInfo{}, // TODO: sensible values?
		},
		config,
		consumer,
	)
}

// TODO: Replace with actual Traces consumer i.e. logic goes here
// The important bit is that this needs to implement the consumer.Traces interface,
// i.e. have a method with signature
//	ConsumeTraces(ctx context.Context, td pdata.Traces) error
// The pdata here                         ^^^^^^^^^^^^
// is the one we use in our exporter in the Collector, not the internal one.
type otlpTracesConsumer struct {
	component.TracesExporter
}

func newConsumer() (otlpTracesConsumer, error) {
	factory := loggingexporter.NewFactory()
	config := factory.CreateDefaultConfig()

	// This example consumer just logs the incoming traces
	exampleConsumer, err := factory.CreateTracesExporter(
		context.TODO(),
		component.ExporterCreateParams{
			Logger:               zapWrapper(),
			ApplicationStartInfo: component.ApplicationStartInfo{}, // TODO: sensible values?
		},
		config,
	)
	if err != nil {
		return otlpTracesConsumer{}, err
	}
	return otlpTracesConsumer{exampleConsumer}, nil
}
