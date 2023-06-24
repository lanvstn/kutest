package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

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

		out[strings.TrimPrefix(e.Name, prefix)] = string(b)
	}

	return out
}
