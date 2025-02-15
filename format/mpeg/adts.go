package mpeg

import (
	"github.com/wader/fq/format"
	"github.com/wader/fq/format/registry"
	"github.com/wader/fq/pkg/decode"
)

var adtsFrame decode.Group

func init() {
	registry.MustRegister(decode.Format{
		Name:        format.ADTS,
		Description: "Audio Data Transport Stream",
		Groups:      []string{format.PROBE},
		DecodeFn:    adtsDecoder,
		RootArray:   true,
		RootName:    "frames",
		Dependencies: []decode.Dependency{
			{Names: []string{format.ADTS_FRAME}, Group: &adtsFrame},
		},
	})
}

func adtsDecoder(d *decode.D, in any) any {
	validFrames := 0
	for !d.End() {
		if dv, _, _ := d.TryFieldFormat("frame", adtsFrame, nil); dv == nil {
			break
		}
		validFrames++
	}

	if validFrames == 0 {
		d.Fatalf("no valid frames")
	}

	return nil
}
