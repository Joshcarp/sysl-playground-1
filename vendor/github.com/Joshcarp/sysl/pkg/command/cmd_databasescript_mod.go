package command

import (
	"fmt"
	"strings"

	"github.com/Joshcarp/sysl/pkg/database"
	"github.com/Joshcarp/sysl/pkg/sysl"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

func GenerateModDatabaseScripts(scriptParams *CmdDatabaseScriptParams, modelOld, modelNew *sysl.Module,
	logger *logrus.Logger) ([]database.ScriptOutput, error) {
	logger.Debugf("Application names: %v\n", scriptParams.appNames)
	logger.Debugf("title: %s\n", scriptParams.title)
	logger.Debugf("outputDir: %s\n", scriptParams.outputDir)
	logger.Debugf("db type: %s\n", scriptParams.dbType)
	appNamesStr := strings.TrimSpace(scriptParams.appNames)
	if appNamesStr == "" {
		logger.Error("no application name specified")
		return nil, fmt.Errorf("no application names specified")
	}
	appNames := strings.Split(appNamesStr, database.Delimiter)
	v := database.MakeDatabaseScriptView(scriptParams.title, logger)
	outputSlice := v.ProcessModSysls(modelOld.GetApps(), modelNew.GetApps(), appNames,
		scriptParams.outputDir, scriptParams.dbType)
	return outputSlice, nil
}

type modDatabaseScriptCmd struct {
	CmdDatabaseScriptParams
}

func (p *modDatabaseScriptCmd) Name() string       { return "generate-db-scripts-delta" }
func (p *modDatabaseScriptCmd) MaxSyslModule() int { return 2 }

func (p *modDatabaseScriptCmd) Configure(app *kingpin.Application) *kingpin.CmdClause {
	cmd := app.Command(p.Name(), "Generate delta db scripts").Alias("generatedbscriptsdelta")

	cmd.Flag("title", "file title").Short('t').StringVar(&p.title)
	cmd.Flag("output-dir", "output directory").Short('o').StringVar(&p.outputDir)
	cmd.Flag("app-names", "application names to read").Short('a').StringVar(&p.appNames)
	cmd.Flag("db-type", "database type e.g postgres").Short('d').StringVar(&p.dbType)
	EnsureFlagsNonEmpty(cmd)
	return cmd
}

func (p *modDatabaseScriptCmd) Execute(args ExecuteArgs) error {
	if len(args.Modules) < 2 {
		return fmt.Errorf("this command needs min 2 module(s)")
	}
	outputSlice, err := GenerateModDatabaseScripts(&p.CmdDatabaseScriptParams,
		args.Modules[0], args.Modules[1], args.Logger)
	if err != nil {
		return err
	}
	return database.GenerateFromSQLMap(outputSlice, args.Filesystem, args.Logger)
}
