package command

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/Joshcarp/sysl/pkg/importer"
	"github.com/spf13/afero"
	"gopkg.in/alecthomas/kingpin.v2"
)

type importCmd struct {
	importer.OutputData
	filename string
	outfile  string
	mode     string
}

func (p *importCmd) Name() string       { return "import" }
func (p *importCmd) MaxSyslModule() int { return 0 }

func (p *importCmd) Configure(app *kingpin.Application) *kingpin.CmdClause {
	opts := []string{modeGrammar, modeSwagger, modeXSD, modeOpenAPI}
	sort.Strings(opts)
	optsText := strings.Join(opts, ", ")
	opts = append(opts, modeAuto)

	cmd := app.Command(p.Name(), "Import foreign type to sysl. Supported types: ["+optsText+"]")
	cmd.Flag("input", "input filename").Short('i').Required().StringVar(&p.filename)
	cmd.Flag("app-name",
		"name of the sysl app to define in sysl model.").Required().Short('a').StringVar(&p.AppName)
	cmd.Flag("package",
		"name of the sysl package to define in sysl model.").Short('p').StringVar(&p.Package)
	cmd.Flag("output", "output filename").Default("output.sysl").Short('o').StringVar(&p.outfile)

	cmd.Flag("format", fmt.Sprintf("format of the input filename, options: [%s, %s]", modeAuto, optsText)).
		Short('f').
		Default(modeAuto).
		HintOptions(opts...).
		EnumVar(&p.mode, opts...)

	EnsureFlagsNonEmpty(cmd, "package")
	return cmd
}

const (
	modeOpenAPI = "openapi"
	modeSwagger = "swagger"
	modeXSD     = "xsd"
	modeGrammar = "grammar"
	modeAuto    = "auto"
)

func (p *importCmd) Execute(args ExecuteArgs) error {
	data, err := ioutil.ReadFile(p.filename)
	if err != nil {
		return err
	}

	if p.mode == "auto" {
		p.mode = guessFileType(p.filename, data)
	}
	var imp importer.Func
	switch p.mode {
	case modeSwagger:
		args.Logger.Infof("Using swagger importer\n")
		imp = importer.LoadSwaggerText
	case modeXSD:
		args.Logger.Infof("Using xsd importer\n")
		imp = importer.LoadXSDText
	case modeGrammar:
		args.Logger.Infof("Using grammar importer\n")
		imp = importer.LoadGrammar
	case modeOpenAPI:
		args.Logger.Infof("Using OpenAPI importer\n")
		imp = importer.LoadOpenAPIText
	default:
		args.Logger.Fatalf("Unsupported input format: %s\n", p.mode)
	}
	output, err := imp(p.OutputData, string(data), args.Logger)
	if err != nil {
		return err
	}
	return afero.WriteFile(args.Filesystem, p.outfile, []byte(output), 0644)
}

func guessYamlType(filename string, data []byte) string {
	for _, check := range []string{modeSwagger, modeOpenAPI} {
		if strings.Contains(string(data), check) {
			return check
		}
	}

	return "unknown"
}

func guessFileType(filename string, data []byte) string {
	parts := strings.Split(filename, ".")
	switch ext := parts[len(parts)-1]; ext {
	case "xml", "xsd":
		return modeXSD
	case "yaml", "yml", "json":
		return guessYamlType(filename, data)
	case "g":
		return modeGrammar
	default:
		return ext
	}
}
