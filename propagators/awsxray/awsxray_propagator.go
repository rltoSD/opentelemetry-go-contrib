// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/api/trace"
)

const (
	traceHeaderKey       = "X-Amzn-Trace-Id"
	traceHeaderDelimiter = ";"
	kvDelimiter          = "="
	traceIdKey           = "Root"
	sampleFlagKey        = "Sampled"
	parentIdKey          = "Parent"
	traceIdVersion       = "1"
	traceIdDelimiter     = "-"
	isSampled            = "1"
	notSampled           = "0"

	traceFlagNone           = 0x0
	traceFlagSampled        = 0x1 << 0
	traceIdLength           = 35
	traceIdDelimitterIndex1 = 1
	traceIdDelimitterIndex2 = 10
	traceIdFirstPartLength  = 8
	parentIdLength          = 16
	sampledFlagLength       = 1
)

var (
	empty                  = trace.EmptySpanContext()
	errInvalidTraceHeader  = errors.New("invalid X-Amzn-Trace-Id header value, should contain 3 different part separated by ;")
	errMalformedTraceID    = errors.New("cannot decode trace id from header, should be a string of hex, lowercase trace id can't be all zero")
	errInvalidSpanIDLength = errors.New("invalid span id length, must be 16")
)

// AwsXray propagator serializes Span Context to/from AWS X-Ray headers
//
// AWS X-Ray format
//
// X-Amzn-Trace-Id: Root={traceId};Parent={parentId};Sampled={samplingFlag}
type AwsXray struct{}

var _ otel.TextMapPropagator = &AwsXray{}

// Inject injects a context to the carrier following AWS X-Ray format.
func (awsxray AwsXray) Inject(ctx context.Context, carrier otel.TextMapCarrier) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	headers := []string{}
	if !sc.TraceID.IsValid() || !sc.SpanID.IsValid() {
		return
	}
	otTraceId := sc.TraceID.String()
	xrayTraceId := traceIdVersion + traceIdDelimiter + otTraceId[0:traceIdFirstPartLength] +
		traceIdDelimiter + otTraceId[traceIdFirstPartLength:]
	parentId := sc.SpanID
	samplingFlag := notSampled
	if sc.TraceFlags == traceFlagSampled {
		samplingFlag = isSampled
	}

	headers = append(headers, traceIdKey, kvDelimiter, xrayTraceId, traceHeaderDelimiter, parentIdKey,
		kvDelimiter, parentId.String(), traceHeaderDelimiter, sampleFlagKey, kvDelimiter, samplingFlag)

	carrier.Set(traceHeaderKey, strings.Join(headers, ""))
}

// Extract extracts a context from the carrier if it contains AWS X-Ray headers.
func (awsxray AwsXray) Extract(ctx context.Context, carrier otel.TextMapCarrier) context.Context {
	// extract tracing information
	if h := carrier.Get(traceHeaderKey); h != "" {
		sc, err := extract(h)
		if err == nil && sc.IsValid() {
			return trace.ContextWithRemoteSpanContext(ctx, sc)
		}
	}
	return ctx
}

func extract(headerVal string) (trace.SpanContext, error) {
	var (
		sc             = trace.SpanContext{}
		err            error
		delimiterIndex int
		part           string
	)
	pos := 0
	for pos < len(headerVal) {
		delimiterIndex = indexOf(headerVal, traceHeaderDelimiter, pos)
		if delimiterIndex >= 0 {
			part = headerVal[pos:delimiterIndex]
			pos = delimiterIndex + 1
		} else {
			//last part
			part = strings.TrimSpace(headerVal[pos:])
			pos = len(headerVal)
		}
		equalsIndex := strings.Index(part, kvDelimiter)
		if equalsIndex < 0 {
			return empty, errInvalidTraceHeader
		}
		value := part[equalsIndex+1:]
		if strings.HasPrefix(part, traceIdKey) {
			sc.TraceID, err = parseTraceId(value)
			if err != nil {
				return empty, errMalformedTraceID
			}
		} else if strings.HasPrefix(part, parentIdKey) {
			//extract parentId
			sc.SpanID, err = trace.SpanIDFromHex(value)
			if err != nil {
				return empty, errInvalidSpanIDLength
			}
		} else if strings.HasPrefix(part, sampleFlagKey) {
			//extract traceflag
			sc.TraceFlags = parseTraceFlag(value)
		}
	}
	return sc, nil
}

//returns position of the first occurence of a substring starting at pos index
func indexOf(str string, substr string, pos int) int {
	index := strings.Index(str[pos:], substr)
	if index > -1 {
		index += pos
	}
	return index
}

//returns trace Id if  valid else return invalid trace Id
func parseTraceId(xrayTraceId string) (trace.ID, error) {
	if len(xrayTraceId) != traceIdLength {
		return empty.TraceID, errMalformedTraceID
	}
	if !strings.HasPrefix(xrayTraceId, traceIdVersion) {
		return empty.TraceID, errMalformedTraceID
	}

	if xrayTraceId[traceIdDelimitterIndex1:traceIdDelimitterIndex1+1] != traceIdDelimiter ||
		xrayTraceId[traceIdDelimitterIndex2:+traceIdDelimitterIndex2+1] != traceIdDelimiter {
		return empty.TraceID, errMalformedTraceID
	}

	epochPart := xrayTraceId[traceIdDelimitterIndex1+1 : traceIdDelimitterIndex2]
	uniquePart := xrayTraceId[traceIdDelimitterIndex2+1 : traceIdLength]

	result := epochPart + uniquePart
	return trace.IDFromHex(result)
}

//returns traceFlag
func parseTraceFlag(xraySampledFlag string) byte {
	if len(xraySampledFlag) == sampledFlagLength && xraySampledFlag != isSampled {
		return traceFlagNone
	}
	return trace.FlagsSampled
}

func (awsxray AwsXray) Fields() []string {
	return []string{traceHeaderKey}
}
