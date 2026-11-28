package d2compiler

import (
	"strings"
	"testing"
)

func TestPopulatedScenarioExpansionBudget(t *testing.T) {
	// Folder detection must not copy the entire inherited board tree a
	// second time. Keep a bounded budget for the actual inheritance, class
	// application and tween sampling in this seven-service diagram.
	const source = `vars: {d2-config: {animate-interval: 200ms; animate-duration: 80ms}; pending: 0}
classes: {
  service: {width: 120; height: 60; style: {stroke: black; font-size: 16}}
  healthy.style.fill: green
  down.style.fill: red
}
checkout; api; primary; relay; broker; worker; ledger: "${pending}"
*.class: [service; healthy]
checkout -> api -> primary -> relay -> broker -> worker -> ledger
scenarios: {
  offline: {broker.class: [service; down]; vars.pending: 100}
  normal: {}
}`
	graph, _, err := Compile("checkout.d2", strings.NewReader(source), &CompileOptions{MaxVariableExpansion: 12_000})
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range graph.Scenarios {
		if len(scenario.TweenFrames) != 2 || scenario.IsFolderOnly {
			t.Fatalf("%s has %d samples, folder=%t", scenario.Name, len(scenario.TweenFrames), scenario.IsFolderOnly)
		}
	}
}

func TestInvalidScenarioStyleReportsAuthoredField(t *testing.T) {
	const source = `vars.d2-config: {animate-duration: 80ms; animate-interval: 120ms}
classes.normal.style.fill: blue
service.class: normal
scenarios: {first: {service.style.fill: red}; second: {service.style: blue}}`
	_, _, err := Compile("styles.d2", strings.NewReader(source), nil)
	if err == nil || !strings.Contains(err.Error(), `"style" expected to be set to a map of key-values`) {
		t.Fatalf("expected the diagnostic for an authored scalar style, got %v", err)
	}
}

func TestQuotedShapeOnBoardWithChildren(t *testing.T) {
	// The layer namespace is quoted on each board so it denotes an object.
	// Empty namespaces and genuinely empty children still consume no slot.
	const source = `"layers".style.fill: blue
scenarios: {
  first: {"layers".style.fill: red}
  second: {"layers".style.fill: green; steps: {empty}}
  third: {"layers".style.fill: yellow}
}`
	graph, _, err := Compile("quoted.d2", strings.NewReader(source), &CompileOptions{
		AnimateTweens: new(true), AnimateDuration: new(int64(80)),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, third := graph.Scenarios[1], graph.Scenarios[2]
	if graph.IsFolderOnly || second.IsFolderOnly || !second.Steps[0].IsFolderOnly {
		t.Fatal("folder detection hid an authored object or exposed an empty board")
	}
	if len(third.TweenFrames) != 2 || third.TweenFrames[0].Objects[0].Style.Fill.Value != "green" {
		t.Fatal("scenario did not start from the drawable sibling")
	}
}
