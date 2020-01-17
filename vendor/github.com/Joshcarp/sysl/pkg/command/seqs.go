package command

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	sysl "github.com/Joshcarp/sysl/pkg/sysl"
	"github.com/Joshcarp/sysl/pkg/syslutil"
	"github.com/sirupsen/logrus"
)

type sequenceDiagParam struct {
	AppLabeler
	EndpointLabeler
	endpoints  []string
	title      string
	blackboxes map[string]*Upto
	appName    string
	group      string
}

func generateSequenceDiag(m *sysl.Module, p *sequenceDiagParam, logger *logrus.Logger) (string, error) {
	w := MakeSequenceDiagramWriter(true, "skinparam maxMessageSize 250")
	v := MakeSequenceDiagramVisitor(p.AppLabeler, p.EndpointLabeler, w, m, p.appName, p.group, logger)
	e := MakeEndpointCollectionElement(p.title, p.endpoints, p.blackboxes)

	if err := e.Accept(v); err != nil {
		return "", err
	}

	const color = "#LightBlue"
	for boxname, appset := range v.groupboxes {
		fmt.Fprintf(w, "box \"%s\" %s\n", boxname, color)
		for key := range appset {
			fmt.Fprintf(w, "\tparticipant %s\n", v.UniqueVarForAppName(key))
		}
		fmt.Fprintf(w, "end box\n")
	}

	return w.String(), nil
}

func constructFormatParser(former, latter string) *FormatParser {
	fmtstr := former
	if former == "" {
		fmtstr = latter
	}

	return MakeFormatParser(escapeWordBoundary(fmtstr))
}

func escapeWordBoundary(src string) string {
	result, err := json.Marshal(src)
	syslutil.PanicOnError(err)
	escapeStr := strings.Replace(string(result), `\u0008`, `\\b`, -1)
	var val string
	err = json.Unmarshal([]byte(escapeStr), &val)
	syslutil.PanicOnError(err)

	return val
}

func DoConstructSequenceDiagrams(
	cmdContextParam *CmdContextParamSeqgen,
	model *sysl.Module,
	logger *logrus.Logger,
) (map[string]string, error) {
	var blackboxes [][]string

	logger.Debugf("endpoints: %v\n", cmdContextParam.endpointsFlag)
	logger.Debugf("app: %v\n", cmdContextParam.appsFlag)
	logger.Debugf("endpoint_format: %s\n", cmdContextParam.endpointFormat)
	logger.Debugf("app_format: %s\n", cmdContextParam.appFormat)
	logger.Debugf("title: %s\n", cmdContextParam.title)
	logger.Debugf("output: %s\n", cmdContextParam.output)

	if len(cmdContextParam.blackboxes) == 0 {
		blackboxes = ParseBlackBoxesFromArgument(cmdContextParam.blackboxesFlag)
		logger.Debugf("blackbox: %s\n", cmdContextParam.blackboxesFlag)
	} else {
		blackboxes = cmdContextParam.blackboxes
	}

	result := make(map[string]string)

	if strings.Contains(cmdContextParam.output, "%(epname)") {
		if len(blackboxes) > 0 {
			logger.Warnf("Ignoring blackboxes passed from command line")
		}
		spout := MakeFormatParser(cmdContextParam.output)
		for _, appName := range cmdContextParam.appsFlag {
			app := model.Apps[appName]
			bbs := TransformBlackBoxes(app.GetAttrs()["blackboxes"].GetA().GetElt())
			spseqtitle := constructFormatParser(app.GetAttrs()["seqtitle"].GetS(), cmdContextParam.title)
			spep := constructFormatParser(app.GetAttrs()["epfmt"].GetS(), cmdContextParam.endpointFormat)
			spapp := constructFormatParser(app.GetAttrs()["appfmt"].GetS(), cmdContextParam.appFormat)
			keys := []string{}
			for k := range app.GetEndpoints() {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			bbsAll := map[string]*Upto{}
			TransformBlackboxesToUptos(bbsAll, bbs, BBApplication)
			var sd *sequenceDiagParam
			for _, k := range keys {
				endpoint := app.GetEndpoints()[k]
				epAttrs := endpoint.GetAttrs()
				outputDir := spout.FmtOutput(appName, k, endpoint.GetLongName(), epAttrs)
				bbs2 := TransformBlackBoxes(endpoint.GetAttrs()["blackboxes"].GetA().GetElt())
				varrefs := MergeAttributes(app.GetAttrs(), endpoint.GetAttrs())
				sdEndpoints := []string{}
				for _, stmt := range endpoint.GetStmt() {
					_, ok := stmt.Stmt.(*sysl.Statement_Call)
					if ok {
						parts := stmt.GetCall().GetTarget().GetPart()
						ep := stmt.GetCall().GetEndpoint()
						sdEndpoints = append(sdEndpoints, strings.Join(parts, " :: ")+" <- "+ep)
					}
				}
				if len(sdEndpoints) == 0 {
					return nil, fmt.Errorf("no call statements to build sequence diagram for endpoint %s",
						endpoint.Name)
				}
				groupAttr := epAttrs["groupby"].GetS()
				if len(groupAttr) == 0 {
					groupAttr = cmdContextParam.group
				} else if len(cmdContextParam.group) > 0 {
					logger.Warnf("Ignoring groupby passed from command line")
				}
				TransformBlackboxesToUptos(bbsAll, bbs2, BBEndpointCollection)
				sd = &sequenceDiagParam{
					endpoints:       sdEndpoints,
					AppLabeler:      spapp,
					EndpointLabeler: spep,
					title:           spseqtitle.FmtSeq(endpoint.GetName(), endpoint.GetLongName(), varrefs),
					blackboxes:      bbsAll,
					appName:         fmt.Sprintf("'%s :: %s'", appName, endpoint.GetName()),
					group:           groupAttr,
				}
				out, err := generateSequenceDiag(model, sd, logger)
				if err != nil {
					return nil, err
				}
				for indx := range bbs2 {
					delete(bbsAll, bbs2[indx][0])
				}
				result[outputDir] = out
			}
			for bbKey, bbVal := range bbsAll {
				if bbVal.VisitCount == 0 && bbVal.ValueType == BBApplication {
					logger.Warnf("blackbox '%s' not hit in app '%s'\n", bbKey, appName)
				}
			}
		}
	} else {
		if len(cmdContextParam.endpointsFlag) == 0 {
			return result, nil
		}
		spep := constructFormatParser("", cmdContextParam.endpointFormat)
		spapp := constructFormatParser("", cmdContextParam.appFormat)
		bbsAll := map[string]*Upto{}
		TransformBlackboxesToUptos(bbsAll, blackboxes, BBCommandLine)
		sd := &sequenceDiagParam{
			endpoints:       cmdContextParam.endpointsFlag,
			AppLabeler:      spapp,
			EndpointLabeler: spep,
			title:           cmdContextParam.title,
			blackboxes:      bbsAll,
			group:           cmdContextParam.group,
		}
		out, err := generateSequenceDiag(model, sd, logger)
		if err != nil {
			return nil, err
		}
		for bbKey, bbVal := range bbsAll {
			if bbVal.VisitCount == 0 && bbVal.ValueType == BBCommandLine {
				logger.Warnf("blackbox '%s' passed on commandline not hit\n", bbKey)
			}
		}
		result[cmdContextParam.output] = out
	}

	return result, nil
}
