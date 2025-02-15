package msgpack

// https://github.com/msgpack/msgpack/blob/master/spec.md

// TODO: ext types done correctly?

import (
	"embed"

	"github.com/wader/fq/format"
	"github.com/wader/fq/format/registry"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/scalar"
)

//go:embed msgpack.jq
var msgPackFS embed.FS

func init() {
	registry.MustRegister(decode.Format{
		Name:        format.MSGPACK,
		Description: "MessagePack",
		DecodeFn:    decodeMsgPack,
		Files:       msgPackFS,
		Functions:   []string{"torepr", "_help"},
	})
}

type formatEntry struct {
	r [2]byte
	s scalar.S
	d func(d *decode.D)
}

type formatEntries []formatEntry

func (fes formatEntries) lookup(u byte) (formatEntry, bool) {
	for _, fe := range fes {
		if u >= fe.r[0] && u <= fe.r[1] {
			return fe, true
		}
	}
	return formatEntry{}, false
}

func (fes formatEntries) MapScalar(s scalar.S) (scalar.S, error) {
	u := s.ActualU()
	if fe, ok := fes.lookup(byte(u)); ok {
		s = fe.s
		s.Actual = u
	}
	return s, nil
}

func decodeMsgPackValue(d *decode.D) {
	arrayFn := func(seekBits int64, lengthBits int) func(d *decode.D) {
		return func(d *decode.D) {
			d.SeekRel(seekBits)
			length := d.FieldU("length", lengthBits)
			d.FieldArray("elements", func(d *decode.D) {
				for i := uint64(0); i < length; i++ {
					d.FieldStruct("element", decodeMsgPackValue)
				}
			})
		}
	}
	mapFn := func(seekBits int64, lengthBits int) func(d *decode.D) {
		return func(d *decode.D) {
			d.SeekRel(seekBits)
			length := d.FieldU("length", lengthBits)
			d.FieldArray("pairs", func(d *decode.D) {
				for i := uint64(0); i < length; i++ {
					d.FieldStruct("pair", func(d *decode.D) {
						d.FieldStruct("key", decodeMsgPackValue)
						d.FieldStruct("value", decodeMsgPackValue)
					})
				}
			})
		}
	}
	extFn := func(lengthBits int) func(d *decode.D) {
		return func(d *decode.D) {
			length := d.FieldU8("length")
			d.FieldS8("fixtype")
			d.FieldRawLen("value", int64(length)*8)
		}
	}

	// is defined here as a global map would cause a init dependency cycle
	formatMap := formatEntries{
		{r: [2]byte{0x00, 0x7f}, s: scalar.S{Sym: "positive_fixint"}, d: func(d *decode.D) {
			d.SeekRel(-8)
			d.FieldU8("value")
		}},
		{r: [2]byte{0x80, 0x8f}, s: scalar.S{Sym: "fixmap"}, d: mapFn(-4, 4)},
		{r: [2]byte{0x90, 0x9f}, s: scalar.S{Sym: "fixarray"}, d: arrayFn(-4, 4)},
		{r: [2]byte{0xa0, 0xbf}, s: scalar.S{Sym: "fixstr"}, d: func(d *decode.D) {
			d.SeekRel(-4)
			length := d.FieldU4("length")
			d.FieldUTF8("value", int(length))
		}},
		{r: [2]byte{0xc0, 0xc0}, s: scalar.S{Sym: "nil"}, d: func(d *decode.D) {
			d.FieldValueNil("value")
		}},
		{r: [2]byte{0xc1, 0xc1}, s: scalar.S{Sym: "never_used"}, d: func(d *decode.D) {
			d.Fatalf("0xc1 never used")
		}},
		{r: [2]byte{0xc2, 0xc2}, s: scalar.S{Sym: "false"}, d: func(d *decode.D) {
			d.FieldValueBool("value", false)
		}},
		{r: [2]byte{0xc3, 0xc3}, s: scalar.S{Sym: "true"}, d: func(d *decode.D) {
			d.FieldValueBool("value", true)
		}},
		{r: [2]byte{0xc4, 0xc4}, s: scalar.S{Sym: "bin8"}, d: func(d *decode.D) { d.FieldRawLen("value", int64(d.FieldU8("length"))*8) }},
		{r: [2]byte{0xc5, 0xc5}, s: scalar.S{Sym: "bin16"}, d: func(d *decode.D) { d.FieldRawLen("value", int64(d.FieldU16("length"))*8) }},
		{r: [2]byte{0xc6, 0xc6}, s: scalar.S{Sym: "bin32"}, d: func(d *decode.D) { d.FieldRawLen("value", int64(d.FieldU32("length"))*8) }},
		{r: [2]byte{0xc7, 0xc7}, s: scalar.S{Sym: "ext8"}, d: extFn(8)},
		{r: [2]byte{0xc8, 0xc8}, s: scalar.S{Sym: "ext16"}, d: extFn(16)},
		{r: [2]byte{0xc9, 0xc9}, s: scalar.S{Sym: "ext32"}, d: extFn(32)},
		{r: [2]byte{0xca, 0xca}, s: scalar.S{Sym: "float32"}, d: func(d *decode.D) { d.FieldF32("value") }},
		{r: [2]byte{0xcb, 0xcb}, s: scalar.S{Sym: "float64"}, d: func(d *decode.D) { d.FieldF64("value") }},
		{r: [2]byte{0xcc, 0xcc}, s: scalar.S{Sym: "uint8"}, d: func(d *decode.D) { d.FieldU8("value") }},
		{r: [2]byte{0xcd, 0xcd}, s: scalar.S{Sym: "uint16"}, d: func(d *decode.D) { d.FieldU16("value") }},
		{r: [2]byte{0xce, 0xce}, s: scalar.S{Sym: "uint32"}, d: func(d *decode.D) { d.FieldU32("value") }},
		{r: [2]byte{0xcf, 0xcf}, s: scalar.S{Sym: "uint64"}, d: func(d *decode.D) { d.FieldU64("value") }},
		{r: [2]byte{0xd0, 0xd0}, s: scalar.S{Sym: "int8"}, d: func(d *decode.D) { d.FieldS8("value") }},
		{r: [2]byte{0xd1, 0xd1}, s: scalar.S{Sym: "int16"}, d: func(d *decode.D) { d.FieldS16("value") }},
		{r: [2]byte{0xd2, 0xd2}, s: scalar.S{Sym: "int32"}, d: func(d *decode.D) { d.FieldS32("value") }},
		{r: [2]byte{0xd3, 0xd3}, s: scalar.S{Sym: "int64"}, d: func(d *decode.D) { d.FieldS64("value") }},
		{r: [2]byte{0xd4, 0xd4}, s: scalar.S{Sym: "fixext1"}, d: func(d *decode.D) { d.FieldS8("fixtype"); d.FieldRawLen("value", 1*8) }},
		{r: [2]byte{0xd5, 0xd5}, s: scalar.S{Sym: "fixext2"}, d: func(d *decode.D) { d.FieldS8("fixtype"); d.FieldRawLen("value", 2*8) }},
		{r: [2]byte{0xd6, 0xd6}, s: scalar.S{Sym: "fixext4"}, d: func(d *decode.D) { d.FieldS8("fixtype"); d.FieldRawLen("value", 4*8) }},
		{r: [2]byte{0xd7, 0xd7}, s: scalar.S{Sym: "fixext8"}, d: func(d *decode.D) { d.FieldS8("fixtype"); d.FieldRawLen("value", 8*8) }},
		{r: [2]byte{0xd8, 0xd8}, s: scalar.S{Sym: "fixext16"}, d: func(d *decode.D) { d.FieldS8("fixtype"); d.FieldRawLen("value", 16*8) }},
		{r: [2]byte{0xd9, 0xd9}, s: scalar.S{Sym: "str8"}, d: func(d *decode.D) { d.FieldUTF8("value", int(d.FieldU8("length"))) }},
		{r: [2]byte{0xda, 0xda}, s: scalar.S{Sym: "str16"}, d: func(d *decode.D) { d.FieldUTF8("value", int(d.FieldU16("length"))) }},
		{r: [2]byte{0xdb, 0xdb}, s: scalar.S{Sym: "str32"}, d: func(d *decode.D) { d.FieldUTF8("value", int(d.FieldU32("length"))) }},
		{r: [2]byte{0xdc, 0xdc}, s: scalar.S{Sym: "array16"}, d: arrayFn(0, 16)},
		{r: [2]byte{0xdd, 0xdd}, s: scalar.S{Sym: "array32"}, d: arrayFn(0, 32)},
		{r: [2]byte{0xde, 0xde}, s: scalar.S{Sym: "map16"}, d: mapFn(0, 16)},
		{r: [2]byte{0xdf, 0xdf}, s: scalar.S{Sym: "map32"}, d: mapFn(0, 32)},
		{r: [2]byte{0xe0, 0xff}, s: scalar.S{Sym: "negative_fixint"}, d: func(d *decode.D) {
			d.SeekRel(-8)
			d.FieldS8("value")
		}},
	}

	typ := d.FieldU8("type", formatMap, scalar.ActualHex)
	if fe, ok := formatMap.lookup(byte(typ)); ok {
		fe.d(d)
	} else {
		panic("unreachable")
	}
}

func decodeMsgPack(d *decode.D, in any) any {
	decodeMsgPackValue(d)
	return nil
}
