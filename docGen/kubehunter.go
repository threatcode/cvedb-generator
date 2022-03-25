package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aquasecurity/avd-generator/docGen/menu"
)

func generateKubeHunterPages(inputPagesDir string, outputPagesDir string) {
	log.Printf("generating kube-hunter pages in: %s...", outputPagesDir)

	if err := os.MkdirAll(outputPagesDir, 0777); err != nil {
		fail(err)
	}
	pages, err := getAllFiles(inputPagesDir)
	if err != nil {
		log.Fatal(err)
	}

	titleRegex := regexp.MustCompile("(?m)title: (.+)$")

	for _, page := range pages {
		b, err := ioutil.ReadFile(page)
		if err != nil {
			log.Println("unable to read original kube hunter doc: ", err)
			continue
		}

		id := strings.ToLower(strings.TrimSuffix(filepath.Base(page), ".md"))
		title := titleRegex.FindSubmatch(b)[1]

		newContent := strings.Replace(string(b), "---", fmt.Sprintf(`---
avd_page_type: avd_page 
icon: kube-hunter
shortName: %s
source: Kube Hunter
aliases: [
	"/kube-hunter/%s"	
]
category: misconfig

remediations:
  - kubernetes

breadcrumbs: 
  - name: Kubernetes
    path: /misconfig/kubernetes


`, string(title), id), 1)
		r := strings.NewReplacer(
			"# {{ page.vid }} - {{ page.title }}", "",
			"vid", "id",
			"categories: ", "types: ",
			"## Remediation", "### Recommended Actions",
			"## References", "### Links",
			"## Issue description", fmt.Sprintf(`
### %s`, string(title)))
		content := r.Replace(newContent)

		err = ioutil.WriteFile(filepath.Join(outputPagesDir, filepath.Base(page)), []byte(content), 0644)
		if err != nil {
			log.Fatalln("unable to write kube hunter page: ", err)
		}
	}

	topLevelPath := filepath.Join(outputPagesDir, "_index.md")
	if err := menu.NewTopLevelMenu("Kube Hunter Misconfiguration", "avd_list", topLevelPath).
		WithHeading("Kube Hunter").
		WithIcon("kube-hunter").
		WithCategory("misconfig").
		WithMenu("Kube Hunter").
		WithMenuID("kubehunter").
		WithMenuParent("kubernetes").
		Generate(); err != nil {
		fail(err)
	}
}
