package etw

import (
	"bytes"
	"encoding/binary"
)

// InType indicates the type of data contained in the ETW event.
type InType byte

// Various InType definitions for TraceLogging. These must match the definitions
// found in TraceLoggingProvider.h in the Windows SDK.
const (
	InTypeNull InType = iota
	InTypeUnicodeString
	InTypeANSIString
	InTypeInt8
	InTypeUint8
	InTypeInt16
	InTypeUint16
	InTypeInt32
	InTypeUint32
	InTypeInt64
	InTypeUint64
	InTypeFloat
	InTypeDouble
	InTypeBool32
	InTypeBinary
	InTypeGUID
	InTypePointerUnsupported
	InTypeFileTime
	InTypeSystemTime
	InTypeSID
	InTypeHexInt32
	InTypeHexInt64
	InTypeCountedString
	InTypeCountedANSIString
	InTypeStruct
	InTypeCountedBinary
)

// OutType specifies a hint to the event decoder for how the value should be
// formatted.
type OutType byte

// Various OutType definitions for TraceLogging. These must match the
// definitions found in TraceLoggingProvider.h in the Windows SDK.
const (
	// OutTypeDefault indicates that the default formatting for the in type will
	// be used by the event decoder.
	OutTypeDefault OutType = iota
	OutTypeNoPrint
	OutTypeString
	OutTypeBoolean
	OutTypeHex
	OutTypePID
	OutTypeTID
	OutTypePort
	OutTypeIPv4
	OutTypeIPv6
	OutTypeSocketAddress
	OutTypeXML
	OutTypeJSON
	OutTypeWin32Error
	OutTypeNTStatus
	OutTypeHResult
	OutTypeFileTime
	OutTypeSigned
	OutTypeUnsigned
	OutTypeUTF8              OutType = 35
	OutTypePKCS7WithTypeInfo OutType = 36
	OutTypeCodePointer       OutType = 37
	OutTypeDateTimeUTC       OutType = 38
)

// EventMetadata maintains a buffer which builds up the metadatadata for an ETW
// event. It needs to be paired with EventData which describes the event.
type EventMetadata struct {
	buffer bytes.Buffer
}

// NewEventMetadata returns a new EventMetadata with event name and initial
// metadata written to the buffer.
func NewEventMetadata(name string) *EventMetadata {
	em := EventMetadata{}
	binary.Write(&em.buffer, binary.LittleEndian, uint16(0)) // Length placeholder
	em.writeTags(0)
	em.buffer.WriteString(name)
	em.buffer.WriteByte(0) // Null terminator for name
	return &em
}

type field struct {
	name    string
	inType  InType
	outType OutType
	tags    uint32
}

func (em *EventMetadata) writeField(f field) {
	em.buffer.WriteString(f.name)
	em.buffer.WriteByte(0) // Null terminator for name

	if f.outType == OutTypeDefault && f.tags == 0 {
		em.buffer.WriteByte(byte(f.inType))
	} else {
		em.buffer.WriteByte(byte(f.inType | 128))
		if f.tags == 0 {
			em.buffer.WriteByte(byte(f.outType))
		} else {
			em.buffer.WriteByte(byte(f.outType | 128))
			em.writeTags(f.tags)
		}
	}
}

func (em *EventMetadata) writeTags(tags uint32) {
	tags &= 0xfffffff

	for {
		val := tags >> 21

		if tags&0x1fffff == 0 {
			em.buffer.WriteByte(byte(val & 0x7f))
			return
		}

		em.buffer.WriteByte(byte(val | 0x80))

		tags <<= 7
	}
}

type fieldOpt func(f *field)

// WithOutType specifies the out type for the field. This value is used as a
// hint by the event decoder for how the field value should be formatted. If no
// out type is specified, a default formatting based on the in type will be
// used.
func WithOutType(outType OutType) fieldOpt {
	return func(f *field) {
		f.outType = outType
	}
}

// WithTags adds a tag to the field. Tags are 28-bit values that have meaning
// only to the event consumer. The top 4 bits of the value will be ignored.
// Multiple uses of this option will cause the tags to be OR'd together.
func WithTags(tags uint32) fieldOpt {
	return func(f *field) {
		f.tags |= tags
	}
}

// AddField appends a single field to the end of the event metadata buffer.
func (em *EventMetadata) AddField(name string, inType InType, opts ...fieldOpt) {
	f := field{
		name:   name,
		inType: inType,
	}

	for _, opt := range opts {
		opt(&f)
	}

	em.writeField(f)
}
