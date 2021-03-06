// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pdata

import (
	"fmt"

	"go.opentelemetry.io/collector/internal"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/protogen/metrics/v1"
)

// MetricsDecoder is an interface to decode bytes into protocol-specific data model.
type MetricsDecoder interface {
	// DecodeMetrics decodes bytes into protocol-specific data model.
	// If the error is not nil, the returned interface cannot be used.
	DecodeMetrics(buf []byte) (interface{}, error)
}

// MetricsEncoder is an interface to encode protocol-specific data model into bytes.
type MetricsEncoder interface {
	// EncodeMetrics encodes protocol-specific data model into bytes.
	// If the error is not nil, the returned bytes slice cannot be used.
	EncodeMetrics(model interface{}) ([]byte, error)
}

// FromMetricsTranslator is an interface to translate pdata.Metrics into protocol-specific data model.
type FromMetricsTranslator interface {
	// FromMetrics translates pdata.Metrics into protocol-specific data model.
	// If the error is not nil, the returned pdata.Metrics cannot be used.
	FromMetrics(md Metrics) (interface{}, error)
}

// ToMetricsTranslator is an interface to translate a protocol-specific data model into pdata.Traces.
type ToMetricsTranslator interface {
	// ToMetrics translates a protocol-specific data model into pdata.Metrics.
	// If the error is not nil, the returned pdata.Metrics cannot be used.
	ToMetrics(src interface{}) (Metrics, error)
}

// MetricsMarshaler marshals pdata.Metrics into bytes.
type MetricsMarshaler interface {
	// Marshal the given pdata.Metrics into bytes.
	// If the error is not nil, the returned bytes slice cannot be used.
	Marshal(td Metrics) ([]byte, error)
}

type metricsMarshaler struct {
	encoder    MetricsEncoder
	translator FromMetricsTranslator
}

// NewMetricsMarshaler returns a new MetricsMarshaler.
func NewMetricsMarshaler(encoder MetricsEncoder, translator FromMetricsTranslator) MetricsMarshaler {
	return &metricsMarshaler{
		encoder:    encoder,
		translator: translator,
	}
}

// Marshal pdata.Metrics into bytes.
func (t *metricsMarshaler) Marshal(td Metrics) ([]byte, error) {
	model, err := t.translator.FromMetrics(td)
	if err != nil {
		return nil, fmt.Errorf("converting pdata to model failed: %w", err)
	}
	buf, err := t.encoder.EncodeMetrics(model)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}
	return buf, nil
}

// MetricsUnmarshaler unmarshalls bytes into pdata.Metrics.
type MetricsUnmarshaler interface {
	// Unmarshal the given bytes into pdata.Metrics.
	// If the error is not nil, the returned pdata.Metrics cannot be used.
	Unmarshal(buf []byte) (Metrics, error)
}

type metricsUnmarshaler struct {
	decoder    MetricsDecoder
	translator ToMetricsTranslator
}

// NewMetricsUnmarshaler returns a new MetricsUnmarshaler.
func NewMetricsUnmarshaler(decoder MetricsDecoder, translator ToMetricsTranslator) MetricsUnmarshaler {
	return &metricsUnmarshaler{
		decoder:    decoder,
		translator: translator,
	}
}

// Unmarshal bytes into pdata.Metrics. On error pdata.Metrics is invalid.
func (t *metricsUnmarshaler) Unmarshal(buf []byte) (Metrics, error) {
	model, err := t.decoder.DecodeMetrics(buf)
	if err != nil {
		return Metrics{}, fmt.Errorf("unmarshal failed: %w", err)
	}
	td, err := t.translator.ToMetrics(model)
	if err != nil {
		return Metrics{}, fmt.Errorf("converting model to pdata failed: %w", err)
	}
	return td, nil
}

// Metrics is an opaque interface that allows transition to the new internal Metrics data, but also facilitates the
// transition to the new components, especially for traces.
//
// Outside of the core repository, the metrics pipeline cannot be converted to the new model since data.MetricData is
// part of the internal package.
type Metrics struct {
	orig *otlpcollectormetrics.ExportMetricsServiceRequest
}

// NewMetrics creates a new Metrics.
func NewMetrics() Metrics {
	return Metrics{orig: &otlpcollectormetrics.ExportMetricsServiceRequest{}}
}

// MetricsFromInternalRep creates Metrics from the internal representation.
// Should not be used outside this module.
func MetricsFromInternalRep(wrapper internal.MetricsWrapper) Metrics {
	return Metrics{orig: internal.MetricsToOtlp(wrapper)}
}

// MetricsFromOtlpProtoBytes converts the OTLP Collector ExportMetricsServiceRequest
// ProtoBuf bytes to Metrics.
//
// Returns an invalid Metrics instance if error is not nil.
func MetricsFromOtlpProtoBytes(data []byte) (Metrics, error) {
	req := otlpcollectormetrics.ExportMetricsServiceRequest{}
	if err := req.Unmarshal(data); err != nil {
		return Metrics{}, err
	}
	return Metrics{orig: &req}, nil
}

// InternalRep returns internal representation of the Metrics.
// Should not be used outside this module.
func (md Metrics) InternalRep() internal.MetricsWrapper {
	return internal.MetricsFromOtlp(md.orig)
}

// ToOtlpProtoBytes converts this Metrics to the OTLP Collector ExportMetricsServiceRequest
// ProtoBuf bytes.
//
// Returns an nil byte-array if error is not nil.
func (md Metrics) ToOtlpProtoBytes() ([]byte, error) {
	return md.orig.Marshal()
}

// Clone returns a copy of MetricData.
func (md Metrics) Clone() Metrics {
	cloneMd := NewMetrics()
	md.ResourceMetrics().CopyTo(cloneMd.ResourceMetrics())
	return cloneMd
}

// ResourceMetrics returns the ResourceMetricsSlice associated with this Metrics.
func (md Metrics) ResourceMetrics() ResourceMetricsSlice {
	return newResourceMetricsSlice(&md.orig.ResourceMetrics)
}

// MetricCount calculates the total number of metrics.
func (md Metrics) MetricCount() int {
	metricCount := 0
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			metricCount += ilm.Metrics().Len()
		}
	}
	return metricCount
}

// OtlpProtoSize returns the size in bytes of this Metrics encoded as OTLP Collector
// ExportMetricsServiceRequest ProtoBuf bytes.
func (md Metrics) OtlpProtoSize() int {
	return md.orig.Size()
}

// MetricAndDataPointCount calculates the total number of metrics and data points.
func (md Metrics) MetricAndDataPointCount() (metricCount int, dataPointCount int) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			metrics := ilm.Metrics()
			metricCount += metrics.Len()
			ms := ilm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				switch m.DataType() {
				case MetricDataTypeIntGauge:
					dataPointCount += m.IntGauge().DataPoints().Len()
				case MetricDataTypeDoubleGauge:
					dataPointCount += m.DoubleGauge().DataPoints().Len()
				case MetricDataTypeIntSum:
					dataPointCount += m.IntSum().DataPoints().Len()
				case MetricDataTypeDoubleSum:
					dataPointCount += m.DoubleSum().DataPoints().Len()
				case MetricDataTypeIntHistogram:
					dataPointCount += m.IntHistogram().DataPoints().Len()
				case MetricDataTypeHistogram:
					dataPointCount += m.Histogram().DataPoints().Len()
				case MetricDataTypeSummary:
					dataPointCount += m.Summary().DataPoints().Len()
				}
			}
		}
	}
	return
}

// MetricDataType specifies the type of data in a Metric.
type MetricDataType int32

const (
	MetricDataTypeNone MetricDataType = iota
	MetricDataTypeIntGauge
	MetricDataTypeDoubleGauge
	MetricDataTypeIntSum
	MetricDataTypeDoubleSum
	MetricDataTypeIntHistogram
	MetricDataTypeHistogram
	MetricDataTypeSummary
)

// String returns the string representation of the MetricDataType.
func (mdt MetricDataType) String() string {
	switch mdt {
	case MetricDataTypeNone:
		return "None"
	case MetricDataTypeIntGauge:
		return "IntGauge"
	case MetricDataTypeDoubleGauge:
		return "DoubleGauge"
	case MetricDataTypeIntSum:
		return "IntSum"
	case MetricDataTypeDoubleSum:
		return "DoubleSum"
	case MetricDataTypeIntHistogram:
		return "IntHistogram"
	case MetricDataTypeHistogram:
		return "Histogram"
	case MetricDataTypeSummary:
		return "Summary"
	}
	return ""
}

// DataType returns the type of the data for this Metric.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DataType() MetricDataType {
	switch ms.orig.Data.(type) {
	case *otlpmetrics.Metric_IntGauge:
		return MetricDataTypeIntGauge
	case *otlpmetrics.Metric_DoubleGauge:
		return MetricDataTypeDoubleGauge
	case *otlpmetrics.Metric_IntSum:
		return MetricDataTypeIntSum
	case *otlpmetrics.Metric_DoubleSum:
		return MetricDataTypeDoubleSum
	case *otlpmetrics.Metric_IntHistogram:
		return MetricDataTypeIntHistogram
	case *otlpmetrics.Metric_DoubleHistogram:
		return MetricDataTypeHistogram
	case *otlpmetrics.Metric_DoubleSummary:
		return MetricDataTypeSummary
	}
	return MetricDataTypeNone
}

// SetDataType clears any existing data and initialize it with an empty data of the given type.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) SetDataType(ty MetricDataType) {
	switch ty {
	case MetricDataTypeIntGauge:
		ms.orig.Data = &otlpmetrics.Metric_IntGauge{IntGauge: &otlpmetrics.IntGauge{}}
	case MetricDataTypeDoubleGauge:
		ms.orig.Data = &otlpmetrics.Metric_DoubleGauge{DoubleGauge: &otlpmetrics.DoubleGauge{}}
	case MetricDataTypeIntSum:
		ms.orig.Data = &otlpmetrics.Metric_IntSum{IntSum: &otlpmetrics.IntSum{}}
	case MetricDataTypeDoubleSum:
		ms.orig.Data = &otlpmetrics.Metric_DoubleSum{DoubleSum: &otlpmetrics.DoubleSum{}}
	case MetricDataTypeIntHistogram:
		ms.orig.Data = &otlpmetrics.Metric_IntHistogram{IntHistogram: &otlpmetrics.IntHistogram{}}
	case MetricDataTypeHistogram:
		ms.orig.Data = &otlpmetrics.Metric_DoubleHistogram{DoubleHistogram: &otlpmetrics.DoubleHistogram{}}
	case MetricDataTypeSummary:
		ms.orig.Data = &otlpmetrics.Metric_DoubleSummary{DoubleSummary: &otlpmetrics.DoubleSummary{}}
	}
}

// IntGauge returns the data as IntGauge.
// Calling this function when DataType() != MetricDataTypeIntGauge will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntGauge() IntGauge {
	return newIntGauge(ms.orig.Data.(*otlpmetrics.Metric_IntGauge).IntGauge)
}

// DoubleGauge returns the data as DoubleGauge.
// Calling this function when DataType() != MetricDataTypeDoubleGauge will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleGauge() DoubleGauge {
	return newDoubleGauge(ms.orig.Data.(*otlpmetrics.Metric_DoubleGauge).DoubleGauge)
}

// IntSum returns the data as IntSum.
// Calling this function when DataType() != MetricDataTypeIntSum  will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntSum() IntSum {
	return newIntSum(ms.orig.Data.(*otlpmetrics.Metric_IntSum).IntSum)
}

// DoubleSum returns the data as DoubleSum.
// Calling this function when DataType() != MetricDataTypeDoubleSum will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) DoubleSum() DoubleSum {
	return newDoubleSum(ms.orig.Data.(*otlpmetrics.Metric_DoubleSum).DoubleSum)
}

// IntHistogram returns the data as IntHistogram.
// Calling this function when DataType() != MetricDataTypeIntHistogram will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) IntHistogram() IntHistogram {
	return newIntHistogram(ms.orig.Data.(*otlpmetrics.Metric_IntHistogram).IntHistogram)
}

// Histogram returns the data as Histogram.
// Calling this function when DataType() != MetricDataTypeHistogram will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) Histogram() Histogram {
	return newHistogram(ms.orig.Data.(*otlpmetrics.Metric_DoubleHistogram).DoubleHistogram)
}

// Summary returns the data as Summary.
// Calling this function when DataType() != MetricDataTypeSummary will cause a panic.
// Calling this function on zero-initialized Metric will cause a panic.
func (ms Metric) Summary() Summary {
	return newSummary(ms.orig.Data.(*otlpmetrics.Metric_DoubleSummary).DoubleSummary)
}

func copyData(src, dest *otlpmetrics.Metric) {
	switch srcData := (src).Data.(type) {
	case *otlpmetrics.Metric_IntGauge:
		data := &otlpmetrics.Metric_IntGauge{IntGauge: &otlpmetrics.IntGauge{}}
		newIntGauge(srcData.IntGauge).CopyTo(newIntGauge(data.IntGauge))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleGauge:
		data := &otlpmetrics.Metric_DoubleGauge{DoubleGauge: &otlpmetrics.DoubleGauge{}}
		newDoubleGauge(srcData.DoubleGauge).CopyTo(newDoubleGauge(data.DoubleGauge))
		dest.Data = data
	case *otlpmetrics.Metric_IntSum:
		data := &otlpmetrics.Metric_IntSum{IntSum: &otlpmetrics.IntSum{}}
		newIntSum(srcData.IntSum).CopyTo(newIntSum(data.IntSum))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleSum:
		data := &otlpmetrics.Metric_DoubleSum{DoubleSum: &otlpmetrics.DoubleSum{}}
		newDoubleSum(srcData.DoubleSum).CopyTo(newDoubleSum(data.DoubleSum))
		dest.Data = data
	case *otlpmetrics.Metric_IntHistogram:
		data := &otlpmetrics.Metric_IntHistogram{IntHistogram: &otlpmetrics.IntHistogram{}}
		newIntHistogram(srcData.IntHistogram).CopyTo(newIntHistogram(data.IntHistogram))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleHistogram:
		data := &otlpmetrics.Metric_DoubleHistogram{DoubleHistogram: &otlpmetrics.DoubleHistogram{}}
		newHistogram(srcData.DoubleHistogram).CopyTo(newHistogram(data.DoubleHistogram))
		dest.Data = data
	case *otlpmetrics.Metric_DoubleSummary:
		data := &otlpmetrics.Metric_DoubleSummary{DoubleSummary: &otlpmetrics.DoubleSummary{}}
		newSummary(srcData.DoubleSummary).CopyTo(newSummary(data.DoubleSummary))
		dest.Data = data
	}
}

// AggregationTemporality defines how a metric aggregator reports aggregated values.
// It describes how those values relate to the time interval over which they are aggregated.
type AggregationTemporality int32

const (
	// AggregationTemporalityUnspecified is the default AggregationTemporality, it MUST NOT be used.
	AggregationTemporalityUnspecified = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_UNSPECIFIED)
	// AggregationTemporalityDelta is an AggregationTemporality for a metric aggregator which reports changes since last report time.
	AggregationTemporalityDelta = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA)
	// AggregationTemporalityCumulative is an AggregationTemporality for a metric aggregator which reports changes since a fixed start time.
	AggregationTemporalityCumulative = AggregationTemporality(otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE)
)

// String returns the string representation of the AggregationTemporality.
func (at AggregationTemporality) String() string {
	return otlpmetrics.AggregationTemporality(at).String()
}
