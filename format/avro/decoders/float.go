package decoders

import (
	"github.com/wader/fq/format/avro/schema"
	"github.com/wader/fq/pkg/decode"
)

func decodeFloatFn(schema schema.SimplifiedSchema) (func(string, *decode.D), error) {
	//a float is written as 4 bytes. The float is converted into a 32-bit integer using a method equivalent to Java's
	//floatToIntBits and then encoded in little-endian format.
	return func(name string, d *decode.D) {
		d.FieldF32(name)
	}, nil
}
