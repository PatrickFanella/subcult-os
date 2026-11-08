package app

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type inputListPrint struct {
	Title    string
	Channels []inputListPrintChannel
	Acts     []inputListPrintAct
}
type inputListPrintAct struct {
	Number int
	actTechSheetDTO
}
type inputListPrintChannel struct {
	Number                                               int
	Acts, Labels, Sources, MicDI, Stands, Phantom, Notes string
}

func distinctTechValues(uses []inputUseDTO, value func(inputUseDTO) string) string {
	seen := map[string]bool{}
	parts := []string{}
	for _, use := range uses {
		text := value(use)
		if text != "" && !seen[text] {
			seen[text] = true
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "; ")
}
func inputListPrintData(list eventInputListDTO) inputListPrint {
	out := inputListPrint{Title: list.EventTitle}
	actNumbers := map[string]string{}
	for i, act := range list.Acts {
		out.Acts = append(out.Acts, inputListPrintAct{Number: i + 1, actTechSheetDTO: act})
		for _, input := range act.Inputs {
			actNumbers[input.ID] = strconv.Itoa(i + 1)
		}
	}
	for _, channel := range list.Channels {
		out.Channels = append(out.Channels, inputListPrintChannel{Number: channel.Number,
			Acts:    distinctTechValues(channel.Uses, func(u inputUseDTO) string { return actNumbers[u.ID] }),
			Labels:  distinctTechValues(channel.Uses, func(u inputUseDTO) string { return u.Label }),
			Sources: distinctTechValues(channel.Uses, func(u inputUseDTO) string { return u.SourceType }),
			MicDI:   distinctTechValues(channel.Uses, func(u inputUseDTO) string { return u.MicDI }),
			Stands:  distinctTechValues(channel.Uses, func(u inputUseDTO) string { return u.Stand }),
			Phantom: distinctTechValues(channel.Uses, func(u inputUseDTO) string {
				if u.PhantomPower {
					return "48V"
				}
				return "Off"
			}),
			Notes: distinctTechValues(channel.Uses, func(u inputUseDTO) string {
				if u.Notes == "" {
					return ""
				}
				if len(channel.Uses) == 1 {
					return u.Notes
				}
				return actNumbers[u.ID] + ": " + u.Notes
			}),
		})
	}
	return out
}

var inputListPrintTemplate = template.Must(template.New("input-list").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="robots" content="noindex,nofollow"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'"><title>Private input list</title><style>
@page{size:A4 portrait;margin:12mm}body{font:9pt/1.2 system-ui,sans-serif;color:#111;background:#fff;margin:0}h1{font-size:15pt;margin:0 0 2mm}p{margin:1mm 0}table{border-collapse:collapse;width:100%;table-layout:fixed;margin:3mm 0}th,td{box-sizing:border-box;border:.2mm solid #777;padding:1mm;vertical-align:top;text-align:left;overflow-wrap:anywhere}th{font-weight:700}thead{display:table-header-group}tr{break-inside:avoid}.channel{width:4%}.act{width:7%}.input{width:15%}.source{width:8%}.mic{width:12%}.stand{width:10%}.phantom{width:7%}.notes{width:37%}.summary{break-inside:avoid;margin-top:2mm}h2{font-size:10pt;margin:2mm 0 1mm}.screen{font-size:10pt;margin-bottom:4mm}@media print{.screen{display:none}}
</style></head><body><main><p class="screen">Private workspace document. Use browser Print / Save as PDF. Keep the exported file private.</p><h1>{{.Title}} — input list</h1><p>Private · act numbers refer to the bill below. Shared inputs use one channel across acts. “48V; Off” means phantom power changes between acts: check before repatching.</p><table><thead><tr><th class="channel">Ch</th><th class="act">Acts</th><th class="input">Input</th><th class="source">Source</th><th class="mic">Mic / DI</th><th class="stand">Stand</th><th class="phantom">48V</th><th class="notes">Notes</th></tr></thead><tbody>{{range .Channels}}<tr><td>{{.Number}}</td><td>{{.Acts}}</td><td>{{.Labels}}</td><td>{{.Sources}}</td><td>{{.MicDI}}</td><td>{{.Stands}}</td><td>{{.Phantom}}</td><td>{{.Notes}}</td></tr>{{end}}</tbody></table>{{range .Acts}}<section class="summary"><h2>{{.Number}}. {{.ActName}}</h2>{{if .MonitorNotes}}<p>Monitors: {{.MonitorNotes}}</p>{{end}}{{if .BacklineBrought}}<p>Brings: {{.BacklineBrought}}</p>{{end}}{{if .BacklineNeeded}}<p>Needs: {{.BacklineNeeded}}</p>{{end}}{{if .StagePlotURL}}<p>Stage plot is available in the authenticated event view.</p>{{end}}</section>{{end}}</main></body></html>`))

func (a *App) writeInputListPrint(w http.ResponseWriter, r *http.Request, list eventInputListDTO) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="event-input-list.html"`)
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'")
	_ = inputListPrintTemplate.Execute(w, inputListPrintData(list))
}
