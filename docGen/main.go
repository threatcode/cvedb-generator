package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aquasecurity/avd-generator/menu"
)

var (
	Years []string

	misConfigurationMenu = menu.New("misconfig", "content/misconfig")
	complianceMenu       = menu.New("compliance", "content/compliance")
)

type Clock interface {
	Now(format ...string) string
}

type realClock struct{}

func (realClock) Now(format ...string) string {
	formatString := time.RFC3339
	if len(format) > 0 {
		formatString = format[0]
	}

	return time.Now().Format(formatString)
}

func main() {

	firstYear := 1999

	for y := firstYear; y <= time.Now().Year(); y++ {
		Years = append(Years, strconv.Itoa(y))
	}

	if err := registerChecks(os.DirFS("../avd-repo/trivy-policies-repo")); err != nil {
		fail(err)
	}

	generateChainBenchPages("../avd-repo/chain-bench-repo/internal/checks", "../avd-repo/content/compliance")
	generateDefsecComplianceSpecPages("../avd-repo/trivy-policies-repo/pkg/compliance", "../avd-repo/content/compliance")
	generateCloudSploitPages("../avd-repo/cloudsploit-repo/plugins", "../avd-repo/content/misconfig", "../avd-repo/remediations-repo/en")
	generateDefsecPages("../avd-repo/trivy-policies-repo/avd_docs", "../avd-repo/content/misconfig")

	nvdGenerator := NewNvdGenerator()
	nvdGenerator.GenerateVulnPages()

	for _, year := range Years {
		nvdGenerator.GenerateReservedPages(year, realClock{})
	}

	createTopLevelMenus()
}

func createTopLevelMenus() {
	if err := menu.NewTopLevelMenu("Misconfiguration", "toplevel_page", "content/misconfig/_index.md").
		WithHeading("Misconfiguration Categories").
		WithIcon("aqua").
		WithCategory("misconfig").Generate(); err != nil {
		fail(err)
	}
	if err := menu.NewTopLevelMenu("Compliance", "toplevel_page", "content/compliance/_index.md").
		WithHeading("Compliance").
		WithIcon("aqua").
		WithCategory("compliance").Generate(); err != nil {
		fail(err)
	}

	if err := misConfigurationMenu.Generate(); err != nil {
		fail(err)
	}
	if err := complianceMenu.Generate(); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Println(err)
	os.Exit(1)
}
