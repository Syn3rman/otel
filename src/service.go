package main

import (
	"context"
	"io"
	"os"
	"math/rand"
	"net/http"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_response_time_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"service"})
)

func newExporter(w io.Writer) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(w),
		stdouttrace.WithPrettyPrint(),
	)
}

func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("demo"),
			semconv.ServiceVersionKey.String("v0.1.0"),
		),
	)
	return r
}

func main() {

	logger := log.New(os.Stdout, "", 0)
	exp, err := newExporter(logger.Writer())
	if err != nil {
				logger.Fatal(err)
	}

	// registry := prometheus.NewRegistry()
	// _ = registry.Register(responseTime)
	prometheus.MustRegister(httpDuration)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(newResource()),
	)
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)

	tracer := tp.Tracer("github.com/Syn3rman/otel/service")

	randomHandler := func(w http.ResponseWriter, req *http.Request) {
		timer := prometheus.NewTimer(httpDuration.WithLabelValues("main service"))

		var span trace.Span
		ctx := context.Background()
		ctx, span = tracer.Start(ctx, "request handler")
		defer span.End()
		span.SetAttributes(attribute.String("http.url", req.URL.Path))
		span.SetAttributes(attribute.String("http.method", req.Method))
		span.SetAttributes(attribute.Int("http.status_code", 200))
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)

		timer.ObserveDuration()

		_, _ = io.WriteString(w, "Handler response\n")
	}

	http.Handle("/metrics", promhttp.Handler())
	go func() {
				_ = http.ListenAndServe(":8000", nil)
	}()

	http.HandleFunc("/", randomHandler)
	go func(){
				_ = http.ListenAndServe(":8000", nil)
	}()
	time.Sleep(100*time.Minute)
}
