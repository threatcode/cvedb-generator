package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/valyala/fastjson"
)

const vulnerabilityPostTemplate = `---
title: "{{.Title}}"
date: {{.Date}}
draft: false
---

### Description
{{.Vulnerability.Description}}

{{ if .Vulnerability.CWEInfo.Name}}
#### Title
{{.Vulnerability.CWEInfo.Name}}
{{end}}

{{- if .Vulnerability.CWEInfo.Description}}
#### Description
{{.Vulnerability.CWEInfo.Description}}
{{end}}

{{- if .Vulnerability.CWEInfo.ExtendedDescription}}
#### Extended Description{{range $ed := .Vulnerability.CWEInfo.ExtendedDescription}}
{{$ed}}{{end}}
{{end}}

{{- if .Vulnerability.CWEInfo.CommonConsequences.Consequence}}
#### Common Consequences{{range $cons := .Vulnerability.CWEInfo.CommonConsequences.Consequence}}
Scope: {{$cons.Scope}}

Impact: {{$cons.Impact}}
{{end}}
{{end}}

{{- if .Vulnerability.CWEInfo.PotentialMitigations.Mitigation}}
#### Potential Mitigations{{range $mitigation := .Vulnerability.CWEInfo.PotentialMitigations.Mitigation}}
{{- if $mitigation.Description}}{{range $d := $mitigation.Description}}
- {{$d}}{{end}}{{end}}{{end}}
{{end}}

{{- if .Vulnerability.CWEInfo.RelatedAttackPatterns.RelatedAttackPattern}}
#### Related Attack Patterns{{range $attack := .Vulnerability.CWEInfo.RelatedAttackPatterns.RelatedAttackPattern}}
- https://cwe.mitre.org/data/definitions/{{$attack.CAPECID}}.html{{end}}
{{end}}

### CVSS
| Version | Vector           | Score  |
| ------------- |:-------------| -----:|
| V2      | {{.Vulnerability.CVSS.V2Vector}} | {{.Vulnerability.CVSS.V2Score}} |
| V3      | {{.Vulnerability.CVSS.V3Vector}} | {{.Vulnerability.CVSS.V3Score}} |

### Dates
- Published: {{.Vulnerability.Dates.Published}}
- Modified: {{.Vulnerability.Dates.Modified}}

### References{{range $element := .Vulnerability.References}}
- {{$element}}{{end}}

<!--- Add Aqua content below --->`

const regoPolicyPostTemplate = `---
title: "{{.Title}}"
date: {{.Date}}
draft: false
---

### {{.Rego.ID}}

### Description
{{.Rego.Description}}

### Severity
{{ .Rego.Severity }}

### Recommended Actions 
{{ .Rego.RecommendedActions }}

### Rego Policy
` + "```\n{{ .Rego.Policy }}\n```" + `
### Links{{range $element := .Rego.Links}}
- {{$element}}{{end}}
`

type Dates struct {
	Published string
	Modified  string
}

type CVSS struct {
	V2Vector string
	V2Score  float64
	V3Vector string
	V3Score  float64
}

type RelatedAttackPattern struct {
	CAPECID int
}

// The RelatedAttackPatternsType complex type contains references to attack patterns associated with this weakness. The association implies those attack patterns may be applicable if an instance of this weakness exists. Each related attack pattern is identified by a CAPEC identifier.
type RelatedAttackPatternsType struct {
	RelatedAttackPattern []RelatedAttackPattern
}

type Mitigation struct {
	Phase       []PhaseEnumeration
	Strategy    MitigationStrategyEnumeration
	Description StructuredTextType
}

// May be one of Policy, Requirements, Architecture and Design, Implementation, Build and Compilation, Testing, Documentation, Bundling, Distribution, Installation, System Configuration, Operation, Patching and Maintenance, Porting, Integration, Manufacturing
type PhaseEnumeration string

// May be one of Attack Surface Reduction, Compilation or Build Hardening, Enforcement by Conversion, Environment Hardening, Firewall, Input Validation, Language Selection, Libraries or Frameworks, Resource Limitation, Output Encoding, Parameterization, Refactoring, Sandbox or Jail, Separation of Privilege
type MitigationStrategyEnumeration string

// The PotentialMitigationsType complex type is used to describe potential mitigations associated with a weakness. It contains one or more Mitigation elements, which each represent individual mitigations for the weakness. The Phase element indicates the development life cycle phase during which this particular mitigation may be applied. The Strategy element describes a general strategy for protecting a system to which this mitigation contributes. The Effectiveness element summarizes how effective the mitigation may be in preventing the weakness. The Description element contains a description of this individual mitigation including any strengths and shortcomings of this mitigation for the weakness.
//
// The optional Mitigation_ID attribute is used by the internal CWE team to uniquely identify mitigations that are repeated across any number of individual weaknesses. To help make sure that the details of these common mitigations stay synchronized, the Mitigation_ID is used to quickly identify those mitigation elements across CWE that should be identical. The identifier is a string and should match the following format: MIT-1.
type PotentialMitigationsType struct {
	Mitigation []Mitigation
}

// The CommonConsequencesType complex type is used to specify individual consequences associated with a weakness. The required Scope element identifies the security property that is violated. The optional Impact element describes the technical impact that arises if an adversary succeeds in exploiting this weakness. The optional Likelihood element identifies how likely the specific consequence is expected to be seen relative to the other consequences. For example, there may be high likelihood that a weakness will be exploited to achieve a certain impact, but a low likelihood that it will be exploited to achieve a different impact. The optional Note element provides additional commentary about a consequence.
//
// The optional Consequence_ID attribute is used by the internal CWE team to uniquely identify examples that are repeated across any number of individual weaknesses. To help make sure that the details of these common examples stay synchronized, the Consequence_ID is used to quickly identify those examples across CWE that should be identical. The identifier is a string and should match the following format: CC-1.
type CommonConsequencesType struct {
	Consequence []Consequence
}

type Consequence struct {
	Scope  []ScopeEnumeration
	Impact []TechnicalImpactEnumeration
}

// May be one of Modify Memory, Read Memory, Modify Files or Directories, Read Files or Directories, Modify Application Data, Read Application Data, DoS: Crash, Exit, or Restart, DoS: Amplification, DoS: Instability, DoS: Resource Consumption (CPU), DoS: Resource Consumption (Memory), DoS: Resource Consumption (Other), Execute Unauthorized Code or Commands, Gain Privileges or Assume Identity, Bypass Protection Mechanism, Hide Activities, Alter Execution Logic, Quality Degradation, Unexpected State, Varies by Context, Reduce Maintainability, Reduce Performance, Reduce Reliability, Other
type TechnicalImpactEnumeration string

// May be one of Confidentiality, Integrity, Availability, Access Control, Accountability, Authentication, Authorization, Non-Repudiation, Other
type ScopeEnumeration string
type StructuredTextType []string

type WeaknessType struct {
	ID                    int
	Name                  string
	Description           string
	PotentialMitigations  PotentialMitigationsType
	RelatedAttackPatterns RelatedAttackPatternsType
	CommonConsequences    CommonConsequencesType
	ExtendedDescription   StructuredTextType
}

type Vulnerability struct {
	ID          string
	CWEID       string
	CWEInfo     WeaknessType
	Description string
	References  []string
	CVSS        CVSS
	Dates       Dates
}

type VulnerabilityPost struct {
	Layout        string
	Title         string
	By            string
	Date          string
	Vulnerability Vulnerability
}

type Rego struct {
	ID                 string
	Description        string
	Links              []string
	Severity           string
	Policy             string
	RecommendedActions string
}

type RegoPost struct {
	Layout string
	Title  string
	By     string
	Date   string
	Rego   Rego
}

func ParseVulnerabilityJSONFile(fileName string) (VulnerabilityPost, error) {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return VulnerabilityPost{}, err
	}

	var vuln Vulnerability
	var p fastjson.Parser
	v, err := p.ParseBytes(b)
	if err != nil {
		return VulnerabilityPost{}, err
	}
	vuln.Description = strings.NewReplacer(`"`, ``, `\`, ``).Replace(string(v.GetStringBytes("cve", "description", "description_data", "0", "value")))
	vuln.ID = string(v.GetStringBytes("cve", "CVE_data_meta", "ID"))
	vuln.CWEID = string(v.GetStringBytes("cve", "problemtype", "problemtype_data", "0", "description", "0", "value"))
	vuln.CVSS = CVSS{
		V2Vector: string(v.GetStringBytes("impact", "baseMetricV2", "cvssV2", "vectorString")),
		V2Score:  v.GetFloat64("impact", "baseMetricV2", "cvssV2", "baseScore"),
		V3Vector: string(v.GetStringBytes("impact", "baseMetricV3", "cvssV3", "vectorString")),
		V3Score:  v.GetFloat64("impact", "baseMetricV3", "cvssV3", "baseScore"),
	}

	publishedDate, _ := time.Parse("2006-01-02T04:05Z", string(v.GetStringBytes("publishedDate")))
	modifiedDate, _ := time.Parse("2006-01-02T04:05Z", string(v.GetStringBytes("lastModifiedDate")))
	vuln.Dates = Dates{
		Published: publishedDate.UTC().Format("2006-01-02T15:04Z"),
		Modified:  modifiedDate.UTC().Format("2006-01-02T15:04Z"),
	}

	var refs []string
	for _, r := range v.GetArray("cve", "references", "reference_data") {
		refs = append(refs, strings.ReplaceAll(r.Get("url").String(), `"`, ``))
	}
	vuln.References = refs

	return VulnerabilityPost{
		Layout:        "vulnerability",
		Title:         vuln.ID,
		By:            "NVD",
		Date:          publishedDate.UTC().Format("2006-01-02 03:04:05 -0700"),
		Vulnerability: vuln,
	}, nil
}

func VulnerabilityPostToMarkdown(blog VulnerabilityPost, outputFile *os.File, customContent string) error {
	t := template.Must(template.New("blog").Parse(vulnerabilityPostTemplate))
	err := t.Execute(outputFile, blog)
	if err != nil {
		return err
	}

	if customContent != "" {
		_, _ = outputFile.WriteString("\n" + customContent)
	}
	return nil
}

func RegoPostToMarkdown(rp RegoPost, outputFile *os.File) error {
	t := template.Must(template.New("regoPost").Parse(regoPolicyPostTemplate))
	err := t.Execute(outputFile, rp)
	if err != nil {
		return err
	}
	return nil
}

func GetAllFiles(dir string) ([]string, error) {
	var filesFound []string
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		filesFound = append(filesFound, file.Name())
	}
	return filesFound, nil
}

func GetAllFilesOfKind(dir string, include string, exclude string) ([]string, error) {
	var filteredFiles []string
	files, err := GetAllFiles(dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if strings.Contains(f, include) && !strings.Contains(f, exclude) {
			filteredFiles = append(filteredFiles, f)
		}
	}
	return filteredFiles, nil
}

func GetCustomContentFromMarkdown(fileName string) string {
	b, _ := ioutil.ReadFile(fileName)

	content := strings.Split(string(b), `<!--- Add Aqua content below --->`)
	switch len(content) {
	case 0, 1:
		return ""
	default:
		return strings.TrimSpace(content[1])
	}
}

func ParseRegoPolicyFile(fileName string) (rp RegoPost, err error) {
	rego, err := ioutil.ReadFile(fileName)
	if err != nil {
		return RegoPost{}, err
	}

	idx := strings.Index(string(rego), "package main")
	metadata := string(rego)[:idx]

	rp.Layout = "regoPolicy"
	rp.By = "Aqua Security"
	rp.Rego.Policy = strings.TrimSpace(string(rego)[idx:])
	rp.Date = time.Unix(1594669401, 0).UTC().String()

	for _, line := range strings.Split(metadata, "\n") {
		r := strings.NewReplacer("@", "", "#", "")
		str := r.Replace(line)
		kv := strings.SplitN(str, ":", 2)
		if len(kv) >= 2 {
			val := strings.TrimSpace(kv[1])
			switch strings.ToLower(strings.TrimSpace(kv[0])) {
			case "id":
				rp.Title = val
			case "description":
				rp.Rego.Description = val
			case "recommended_actions":
				rp.Rego.RecommendedActions = val
			case "severity":
				rp.Rego.Severity = val
			case "title":
				rp.Rego.ID = val
				// TODO: Add case for parsing links
			}
		}
	}

	return
}

func main() {
	generateVulnPages()
	generateRegoPages()
}

func generateVulnPages() {
	years := []string{
		"1999", "2000", "2001", "2002", "2003", "2004", "2005",
		"2006", "2007", "2008", "2009", "2010",
		"2011", "2012", "2013", "2014", "2015",
		"2016", "2017", "2018", "2019",
		"2020",
	}

	var wg sync.WaitGroup
	for _, year := range years {
		year := year
		wg.Add(1)

		log.Printf("generating vuln year: %s\n", year)
		//nvdDir := fmt.Sprintf("goldens/json")
		//postsDir := "temp-posts"
		nvdDir := fmt.Sprintf("vuln-list/nvd/%s/", year)
		postsDir := "content/nvd"
		cweDir := fmt.Sprintf("vuln-list/cwe")

		go func(year string) {
			defer wg.Done()
			generateVulnerabilityPages(nvdDir, cweDir, postsDir)
		}(year)
	}
	wg.Wait()
}

func generateRegoPages() {
	for _, p := range []string{"kubernetes"} {
		policyDir := filepath.Join("appshield-repo", "policies", p, "policy")
		log.Printf("generating policies in: %s...", policyDir)
		generateRegoPolicyPages(policyDir, "content/appshield")
	}
}

func generateVulnerabilityPages(nvdDir string, cweDir string, postsDir string) {
	files, err := GetAllFiles(nvdDir)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		bp, err := ParseVulnerabilityJSONFile(filepath.Join(nvdDir, file))
		if err != nil {
			log.Printf("unable to parse file: %s, err: %s, skipping...\n", file, err)
			continue
		}

		_ = AddCWEInformation(&bp, cweDir)

		// check if file exists first, if does then open, if not create
		f, err := os.OpenFile(filepath.Join(postsDir, fmt.Sprintf("%s.md", bp.Title)), os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("unable to create file: %s for markdown, err: %s, skipping...\n", file, err)
			continue
		}

		customContent := GetCustomContentFromMarkdown(f.Name())
		if customContent != "" {
			_ = os.Truncate(f.Name(), 0) // truncate file if custom data was found
		}
		if err := VulnerabilityPostToMarkdown(bp, f, customContent); err != nil {
			log.Printf("unable to write file: %s as markdown, err: %s, skipping...\n", file, err)
			continue
		}
		_ = f.Close()
	}
}

func AddCWEInformation(bp *VulnerabilityPost, cweDir string) error {
	b, err := ioutil.ReadFile(filepath.Join(cweDir, fmt.Sprintf("%s.json", bp.Vulnerability.CWEID)))
	if err != nil {
		return err
	}

	var w WeaknessType
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}

	bp.Vulnerability.CWEInfo = w
	return nil
}

func generateRegoPolicyPages(policyDir string, postsDir string) {
	files, err := GetAllFilesOfKind(policyDir, "rego", "_test")

	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		rp, err := ParseRegoPolicyFile(filepath.Join(policyDir, file))
		if err != nil {
			log.Printf("unable to parse file: %s, err: %s, skipping...\n", file, err)
			continue
		}

		f, err := os.Create(filepath.Join(postsDir, fmt.Sprintf("%s.md", rp.Title)))
		if err != nil {
			log.Printf("unable to create file: %s for markdown, err: %s, skipping...\n", file, err)
			continue
		}
		if err := RegoPostToMarkdown(rp, f); err != nil {
			log.Printf("unable to write file: %s as markdown, err: %s, skipping...\n", file, err)
			continue
		}
		_ = f.Close()
	}
}