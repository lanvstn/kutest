package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/robert-nix/ansihtml"
)

// Yes, this package uses text/template instead of html/template.
// Your test code is assumed to not be a malicious input.
// This converter is anyways only a starting point and
// should be rewritten for serious business is to be done.

//go:embed template.html
var htmlTemplate []byte

func main() {
	t := template.New("report")

	t.Funcs(template.FuncMap{
		"displayLogs": displayLogs,
	})

	_, err := t.Parse(string(htmlTemplate))
	if err != nil {
		log.Fatalf("parse template: %v", err)
	}

	var ginkgoReports []ginkgo.Report
	err = json.NewDecoder(os.Stdin).Decode(&ginkgoReports)
	if err != nil {
		log.Fatalf("parse report: %v", err)
	}

	t.Execute(os.Stdout, ginkgoReports)

}

func displayLogs(re []types.ReportEntry) map[string]string {
	const prefix = "kutest-log-b64-"
	out := make(map[string]string)
	for _, e := range re {
		if !strings.HasPrefix(e.Name, prefix) {
			continue
		}

		s := e.Value.Representation
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			panic(fmt.Sprintf("invalid base64 in log entry! %v", err))
		}

		out[strings.TrimPrefix(e.Name, prefix)] = string(ansihtml.ConvertToHTML(b))
	}

	return out
}
