//go:build fuzz

package format_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/wader/fq/format/all"
	"github.com/wader/fq/format/registry"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/interp"
)

type fuzzFS struct{}

func (fuzzFS) Open(name string) (fs.File, error) {
	return nil, fmt.Errorf("%s: file not found", name)
}

type fuzzTest struct {
	b []byte
	f decode.Format
}

type fuzzTestInput struct {
	interp.FileReader
	io.Writer
}

func (fuzzTestInput) IsTerminal() bool { return false }
func (fuzzTestInput) Size() (int, int) { return 120, 25 }

type fuzzTestOutput struct {
	io.Writer
}

func (o fuzzTestOutput) Size() (int, int) { return 120, 25 }
func (o fuzzTestOutput) IsTerminal() bool { return false }

func (ft *fuzzTest) Platform() interp.Platform { return interp.Platform{} }
func (ft *fuzzTest) Stdin() interp.Input {
	return fuzzTestInput{FileReader: interp.FileReader{R: bytes.NewBuffer(ft.b)}}
}
func (ft *fuzzTest) Stdout() interp.Output        { return fuzzTestOutput{ioutil.Discard} }
func (ft *fuzzTest) Stderr() interp.Output        { return fuzzTestOutput{ioutil.Discard} }
func (ft *fuzzTest) InterruptChan() chan struct{} { return nil }
func (ft *fuzzTest) Environ() []string            { return nil }
func (ft *fuzzTest) Args() []string {
	return []string{
		`fq`,
		`-d`, ft.f.Name,
		`.`,
	}
}
func (ft *fuzzTest) ConfigDir() (string, error) { return "/config", nil }
func (ft *fuzzTest) FS() fs.FS                  { return fuzzFS{} }
func (ft *fuzzTest) History() ([]string, error) { return nil, nil }

func (ft *fuzzTest) Readline(opts interp.ReadlineOpts) (string, error) {
	return "", io.EOF
}

func FuzzFormats(f *testing.F) {
	i := 0

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if filepath.Base(path) != "testdata" {
			return nil
		}

		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if filepath.Ext(path) == ".fqtest" {
				return nil
			}
			if st, err := os.Stat(path); err != nil || st.IsDir() {
				return nil
			}

			b, err := ioutil.ReadFile(path)
			if err != nil {
				f.Fatal(err)
			}

			f.Logf("seed#%d %s", i, path)
			f.Add(b)
			i++

			return nil
		})
		return nil
	})

	gi := 0
	g := registry.Default.MustAll()

	f.Fuzz(func(t *testing.T, b []byte) {
		fz := &fuzzTest{b: b, f: g[gi]}
		q, err := interp.New(fz, registry.Default)
		if err != nil {
			t.Fatal(err)
		}

		_ = q.Main(context.Background(), fz.Stdout(), "dev")
		// if err != nil {
		// 	// TODO: expect error
		// 	t.Fatal(err)
		// }

		gi = (gi + 1) % len(g)
	})
}
