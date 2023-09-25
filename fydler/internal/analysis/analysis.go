package analysis

import (
	"encoding/json"
	"flag"
	"io"
	"strconv"

	"github.com/andrewkroh/go-fleetpkg"
)

type Analyzer struct {
	Name        string
	Description string
	Requires    []*Analyzer

	Flags flag.FlagSet

	Run func(*Pass) (interface{}, error)
}

type Pass struct {
	Analyzer *Analyzer

	// Field information.
	Fields []*fleetpkg.Field // Fields from every file.
	Flat   []*fleetpkg.Field // Flat view of all fields sorted by file and line number.

	// ResultOf provides the inputs to this analysis pass, which are
	// the corresponding results of its prerequisite analyzers.
	// The map keys are the elements of Analysis.Required,
	// and the type of each corresponding value is the required
	// analysis's ResultType.
	ResultOf map[*Analyzer]interface{}

	Report func(Diagnostic)
}

type Pos struct {
	File string
	Line int
	Col  int
}

func NewPos(meta fleetpkg.FileMetadata) Pos {
	return Pos{
		File: meta.Path(),
		Line: meta.Line(),
		Col:  meta.Column(),
	}
}

func (p Pos) String() string {
	if p.Col == 0 {
		return p.File + ":" + strconv.Itoa(p.Line)
	}
	return p.File + ":" + strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Col)
}

func (p Pos) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

type Diagnostic struct {
	Pos      Pos
	Category string
	Message  string
	Related  []RelatedInformation `json:"Related,omitempty"`
}

type RelatedInformation struct {
	Pos     Pos
	Message string
}

type Printer func(diags []Diagnostic, w io.Writer)
