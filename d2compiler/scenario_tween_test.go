package d2compiler

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/d2lang/d2/d2graph"
	"github.com/d2lang/d2/d2target"
	"github.com/d2lang/d2/lib/animation"
)

func TestPopulatedScenarioExpansionBudget(t *testing.T) {
	// Folder detection must not copy the entire inherited board tree a
	// second time. Keep a bounded budget for the actual inheritance, class
	// application and tween sampling in this seven-service diagram.
	const source = `vars: {d2-config: {animate-interval: 200ms; animate-duration: 32ms}; pending: 0}
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
		AnimateTweens: new(true), AnimateDuration: new(int64(32)),
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

func TestScenarioPinsRetainFractionalMotion(t *testing.T) {
	const source = `vars: {d2-config: {animate-interval: 1200ms; animate-duration: 1s}; y: 10}
marker: {left: 0; top: ${y}}
scenarios: {
  move: {marker.left: 5; vars.y: 5}
  reset: {}
}`
	g, _, err := Compile("pins.d2", strings.NewReader(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Scenarios) != 2 {
		t.Fatalf("got %d scenarios, want two", len(g.Scenarios))
	}
	for i, board := range g.Scenarios {
		if n := len(board.TweenFrames); n != d2target.TweenFrameCount(1000) {
			t.Fatalf("%s has %d samples", board.Name, n)
		}
		for j, frame := range board.TweenFrames {
			distance := 5 * animation.EaseInOut(float64(j)/float64(len(board.TweenFrames)))
			if i == 1 {
				distance = 5 - distance
			}
			marker := frame.Objects[0]
			for _, axis := range []struct {
				pin  *d2graph.Scalar
				want float64
			}{{marker.Left, distance}, {marker.Top, 10 - distance}} {
				if axis.pin == nil {
					t.Fatalf("%s sample %d lost its pin", board.Name, j)
				}
				value, err := strconv.Atoi(axis.pin.Value)
				if err != nil || !axis.pin.TweenedPin || math.Abs(float64(value)+axis.pin.TweenOffset-axis.want) > 1e-9 {
					t.Fatalf("%s sample %d pin = %+v, want %g", board.Name, j, axis.pin, axis.want)
				}
			}
		}
	}
}
