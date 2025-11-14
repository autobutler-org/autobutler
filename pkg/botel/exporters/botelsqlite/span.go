package botelsqlite

import (
	"fmt"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanToJSON converts a ReadOnlySpan to the JSON structure shown in types.go
func SpanToJSON(span sdktrace.ReadOnlySpan) (map[string]any, error) {
	spanCtx := span.SpanContext()
	parentCtx := span.Parent()

	result := map[string]any{
		"Name": span.Name(),
		"SpanContext": map[string]any{
			"TraceID":    spanCtx.TraceID().String(),
			"SpanID":     spanCtx.SpanID().String(),
			"TraceFlags": fmt.Sprintf("%02x", spanCtx.TraceFlags()),
			"TraceState": spanCtx.TraceState().String(),
			"Remote":     spanCtx.IsRemote(),
		},
		"Parent": map[string]any{
			"TraceID":    parentCtx.TraceID().String(),
			"SpanID":     parentCtx.SpanID().String(),
			"TraceFlags": fmt.Sprintf("%02x", parentCtx.TraceFlags()),
			"TraceState": parentCtx.TraceState().String(),
			"Remote":     parentCtx.IsRemote(),
		},
		"SpanKind":          span.SpanKind(),
		"StartTime":         span.StartTime(),
		"EndTime":           span.EndTime(),
		"DroppedAttributes": span.DroppedAttributes(),
		"DroppedEvents":     span.DroppedEvents(),
		"DroppedLinks":      span.DroppedLinks(),
		"ChildSpanCount":    span.ChildSpanCount(),
	}

	// Convert attributes
	attrs := []map[string]any{}
	for _, attr := range span.Attributes() {
		attrs = append(attrs, map[string]any{
			"Key": string(attr.Key),
			"Value": map[string]any{
				"Type":  attr.Value.Type().String(),
				"Value": attr.Value.AsInterface(),
			},
		})
	}
	result["Attributes"] = attrs

	// Convert events
	events := []map[string]any{}
	for _, event := range span.Events() {
		eventAttrs := []map[string]any{}
		for _, attr := range event.Attributes {
			eventAttrs = append(eventAttrs, map[string]any{
				"Key": string(attr.Key),
				"Value": map[string]any{
					"Type":  attr.Value.Type().String(),
					"Value": attr.Value.AsInterface(),
				},
			})
		}
		events = append(events, map[string]any{
			"Name":       event.Name,
			"Time":       event.Time,
			"Attributes": eventAttrs,
		})
	}
	if len(events) == 0 {
		result["Events"] = nil
	} else {
		result["Events"] = events
	}

	// Convert links
	links := []map[string]any{}
	for _, link := range span.Links() {
		linkAttrs := []map[string]any{}
		for _, attr := range link.Attributes {
			linkAttrs = append(linkAttrs, map[string]any{
				"Key": string(attr.Key),
				"Value": map[string]any{
					"Type":  attr.Value.Type().String(),
					"Value": attr.Value.AsInterface(),
				},
			})
		}
		links = append(links, map[string]any{
			"SpanContext": map[string]any{
				"TraceID":    link.SpanContext.TraceID().String(),
				"SpanID":     link.SpanContext.SpanID().String(),
				"TraceFlags": fmt.Sprintf("%02x", link.SpanContext.TraceFlags()),
				"TraceState": link.SpanContext.TraceState().String(),
				"Remote":     link.SpanContext.IsRemote(),
			},
			"Attributes": linkAttrs,
		})
	}
	if len(links) == 0 {
		result["Links"] = nil
	} else {
		result["Links"] = links
	}

	// Status
	status := span.Status()
	result["Status"] = map[string]any{
		"Code":        status.Code.String(),
		"Description": status.Description,
	}

	// Check status code explicitly
	switch status.Code {
	case codes.Unset:
		result["Status"].(map[string]any)["Code"] = "Unset"
	case codes.Ok:
		result["Status"].(map[string]any)["Code"] = "Ok"
	case codes.Error:
		result["Status"].(map[string]any)["Code"] = "Error"
	}

	// Resource attributes
	resourceAttrs := []map[string]any{}
	for _, attr := range span.Resource().Attributes() {
		resourceAttrs = append(resourceAttrs, map[string]any{
			"Key": string(attr.Key),
			"Value": map[string]any{
				"Type":  attr.Value.Type().String(),
				"Value": attr.Value.AsInterface(),
			},
		})
	}
	result["Resource"] = resourceAttrs

	// Instrumentation scope
	scope := span.InstrumentationScope()
	scopeAttrs := []map[string]any{}
	iter := scope.Attributes.Iter()
	for iter.Next() {
		attr := iter.Attribute()
		scopeAttrs = append(scopeAttrs, map[string]any{
			"Key": string(attr.Key),
			"Value": map[string]any{
				"Type":  attr.Value.Type().String(),
				"Value": attr.Value.AsInterface(),
			},
		})
	}
	if len(scopeAttrs) == 0 {
		scopeAttrs = nil
	}

	result["InstrumentationScope"] = map[string]any{
		"Name":       scope.Name,
		"Version":    scope.Version,
		"SchemaURL":  scope.SchemaURL,
		"Attributes": scopeAttrs,
	}

	// For backward compatibility
	result["InstrumentationLibrary"] = result["InstrumentationScope"]

	return result, nil
}
