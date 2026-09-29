// Package otel — Kafka span exporter for kvx.
//
// 将 OTel span 序列化为 JSON 后写入 Kafka topic，对齐 cachex/otel 上报格式。
package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// kafkaExporter 实现 sdktrace.SpanExporter 接口，将 span 写入 Kafka
type kafkaExporter struct {
	brokers []string
	topic   string
	w       *kafka.Writer
}

// newMQExporter 创建 Kafka span 导出器
func newMQExporter(brokerURL, topic string) (*kafkaExporter, error) {
	w := &kafka.Writer{
		Addr:         kafka.TCP(parseBrokers(brokerURL)...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		Async:        false,
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 10 * time.Millisecond,
	}
	return &kafkaExporter{brokers: parseBrokers(brokerURL), topic: topic, w: w}, nil
}

// parseBrokers 将逗号分隔的 broker 字符串拆分为 []string
func parseBrokers(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if part := trimSpace(s[start:i]); part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

// trimSpace 去除首尾空白字符
func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for i < j && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

// ExportSpans 批量导出 spans（实现 sdktrace.SpanExporter 接口）
//
// span 参数类型为 []sdktrace.ReadOnlySpan（非 ReadOnlySpans，后者仅在部分第三方包中存在别名）
func (e *kafkaExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, s := range spans {
		msg := kafka.Message{
			Key:   []byte(s.SpanContext().TraceID().String()),
			Value: spanToJSON(s),
			Headers: []kafka.Header{
				{Key: "service", Value: []byte("kvx")},
				{Key: "name", Value: []byte(s.Name())},
			},
		}
		if err := e.w.WriteMessages(ctx, msg); err != nil {
			return fmt.Errorf("otel/mq: write to kafka: %w", err)
		}
	}
	return nil
}

// Shutdown 优雅关闭 Kafka 生产者，实现 sdktrace.SpanExporter.Shutdown
func (e *kafkaExporter) Shutdown(_ context.Context) error {
	if e.w != nil {
		return e.w.Close()
	}
	return nil
}

// spanToJSON 将 ReadOnlySpan 序列化为 JSON 字节（对齐 cachex/otel 格式）
func spanToJSON(s sdktrace.ReadOnlySpan) []byte {
	attrs := make(map[string]string, len(s.Attributes()))
	for _, kv := range s.Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}

	data := map[string]interface{}{
		"name":            s.Name(),
		"trace_id":        s.SpanContext().TraceID().String(),
		"span_id":         s.SpanContext().SpanID().String(),
		"parent_span_id":  s.Parent().SpanID().String(),
		"start_time":      s.StartTime().UTC().Format(time.RFC3339Nano),
		"end_time":        s.EndTime().UTC().Format(time.RFC3339Nano),
		"duration_us":     s.EndTime().Sub(s.StartTime()).Microseconds(),
		"span_kind":       s.SpanKind().String(),
		"instrumentation": s.InstrumentationScope().Name,
		"status":          s.Status().Code.String(),
		"attributes":      attrs,
		"events":          eventListToMap(s.Events()),
	}
	b, _ := json.Marshal(data)
	return b
}

// eventListToMap 将 sdktrace.Event 列表转为 []map，供 JSON 序列化
func eventListToMap(events []sdktrace.Event) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(events))
	for _, ev := range events {
		evAttrs := make(map[string]interface{}, len(ev.Attributes))
		for _, kv := range ev.Attributes {
			evAttrs[string(kv.Key)] = kv.Value.AsString()
		}
		out = append(out, map[string]interface{}{
			"name":       ev.Name,
			"timestamp":  ev.Time.UTC().Format(time.RFC3339Nano),
			"attributes": evAttrs,
		})
	}
	return out
}
