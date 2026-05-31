package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func initTracer() *sdktrace.TracerProvider {
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint("127.0.0.1:4318"),
	)
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("service-a"),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return tp
}

func main() {
	tp := initTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:18081/calculate", nil)
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"orderId":123,"status":"PRICE_CALCULATED","calculation":%s}`, string(body))))
	})

	http.Handle("/order", otelhttp.NewHandler(handler, "order-handler"))

	log.Println("Service A listening on :18080")
	log.Fatal(http.ListenAndServe(":18080", nil))
}
