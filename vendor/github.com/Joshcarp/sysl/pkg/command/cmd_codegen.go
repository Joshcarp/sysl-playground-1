package command

import (
	"fmt"

	"github.com/Joshcarp/sysl/pkg/eval"
	"github.com/Joshcarp/sysl/pkg/syslutil"
	"github.com/Joshcarp/sysl/pkg/validate"
	"gopkg.in/alecthomas/kingpin.v2"
)

type codegenCmd struct {
	CmdContextParamCodegen
	outDir         string
	appName        string
	validateOnly   bool
	enableDebugger bool
}

func (p *codegenCmd) Name() string       { return "codegen" }
func (p *codegenCmd) MaxSyslModule() int { return 1 }

func (p *codegenCmd) Configure(app *kingpin.Application) *kingpin.CmdClause {
	cmd := app.Command(p.Name(), "Generate code").Alias("gen")
	cmd.Flag("root-transform",
		"sysl root directory for input transform file (default: .)").
		Default(".").StringVar(&p.rootTransform)
	cmd.Flag("transform", "path to transform file from the root transform directory").Required().StringVar(&p.transform)
	cmd.Flag("grammar", "path to grammar file").Required().StringVar(&p.grammar)
	cmd.Flag("app-name",
		"name of the sysl app defined in sysl model."+
			" if there are multiple apps defined in sysl model,"+
			" code will be generated only for the given app").Default("").StringVar(&p.appName)
	cmd.Flag("start", "start rule for the grammar").Default(".").StringVar(&p.start)
	cmd.Flag("outdir", "output directory").Default(".").StringVar(&p.outDir)
	cmd.Flag("validate-only", "Only Perform validation on the transform grammar").BoolVar(&p.validateOnly)
	cmd.Flag("disable-validator", "Disable validation on the transform grammar").
		Default("false").BoolVar(&p.disableValidator)
	cmd.Flag("debugger", "Enable the evaluation debugger on error").Default("false").BoolVar(&p.enableDebugger)
	EnsureFlagsNonEmpty(cmd, "app-name")
	return cmd
}

func (p *codegenCmd) Execute(args ExecuteArgs) error {
	if p.validateOnly {
		return validate.DoValidate(validate.Params{
			RootTransform: p.rootTransform,
			Transform:     p.transform,
			Grammar:       p.grammar,
			Start:         p.start,
			Filesystem:    args.Filesystem,
			Logger:        args.Logger,
		})
	}
	if p.appName == "" {
		if len(args.Modules[0].Apps) > 1 {
			args.Logger.Errorf("required argument --app-name value missing")
			return fmt.Errorf("missing required argument")
		}
		p.appName = args.DefaultAppName
	}
	eval.EnableDebugger = p.enableDebugger
	output, err := GenerateCode(&p.CmdContextParamCodegen, args.Modules[0], p.appName, args.Filesystem, args.Logger)
	if err != nil {
		return err
	}
	return outputToFiles(output, syslutil.NewChrootFs(args.Filesystem, p.outDir))
}
