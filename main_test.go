package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

var testUrl string

var testUrlContent = "<a href=\"/some-page\">"

func TestMain(m *testing.M) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, testUrlContent)
	}))
	defer ts.Close()
	testUrl = ts.URL

	retCode := m.Run()

	os.Exit(retCode)
}

func TestParseFlags(t *testing.T) {
	os.Args = []string{"cmd", "some-url", "output-path"}

	baseUrl, outputPath, _ := parseFlags()

	if baseUrl != "some-url" {
		t.Errorf("Expected some-url, got %s", baseUrl)
	}

	if outputPath != "output-path" {
		t.Errorf("expected output-path, got %s", outputPath)
	}
}

func TestParseFlagsNotRightNumberOfArguments(t *testing.T) {
	os.Args = []string{"cmd", "some-url"}

	_, _, err := parseFlags()

	if err == nil {
		t.Error("Expected error, none returned")
	}
}

func TestGenerateGraphDotDefinition(t *testing.T) {
	edges := []edge{
		{
			source: "a",
			dest:   "b",
		},
		{
			source: "b",
			dest:   "c",
		},
	}

	result := generateGraphDotDefinition(edges)

	expected := `digraph Sitemap {
	"a" -> "b";
	"b" -> "c";
}`

	if result != expected {
		t.Error("Resulting dot definition doesn't match expectation")
	}
}

func TestGetUrlContents(t *testing.T) {
	result, _ := getUrlContents(testUrl)

	if strings.TrimSpace(result) != testUrlContent {
		t.Errorf("Unexpected result, got %s, expected %s", result, testUrlContent)
	}
}

func TestGetUrlContentsNonExistingUrl(t *testing.T) {
	result, err := getUrlContents("https://some-random-url-that-really-does-not-exist.coffee")

	if err == nil {
		t.Error("Expected error was not returned")
	}

	if result != "" {
		t.Error("Result should be empty")
	}
}

func TestGetLinksInUrl(t *testing.T) {
	result := getAllLinksInUrl(testUrl)

	if result[0] != "/some-page" {
		t.Error("Returned unexpected result")
	}
}

func TestGetLinksInUrlThatDoesNotExist(t *testing.T) {
	result := getAllLinksInUrl("https://really-should-not-exist.coffee")

	if len(result) != 0 {
		t.Error("Result should be empty")
	}
}

func TestCoordinatorRoutine(t *testing.T) {
	someUrl := "https://some.url"

	workChannel, edgeChannel, resultChannel := startSiteMapBuilderCoordinator(someUrl, 5 * time.Millisecond)

	edgeChannel <- edge{
		source: "https://some.url/",
		dest:   "/page",
	}
	edgeChannel <- edge{
		source: "https://some.url/page",
		dest:   "/page2",
	}
	edgeChannel <- edge{
		source: "https://some.url/page2",
		dest:   "/",
	}

	resultingGraph := <-resultChannel
	resultingWorkUnits := make([]string, 0)
	for workUnit := range workChannel {
		resultingWorkUnits = append(resultingWorkUnits, workUnit)
	}

	expectedGraph := []edge{
		{
			source: "/",
			dest:   "/page",
		},
		{
			source: "/page",
			dest:   "/page2",
		},
		{
			source: "/page2",
			dest:   "/",
		},
	}

	expectedWorkUnits := []string{
		"https://some.url/",
		"https://some.url/page",
		"https://some.url/page2",
	}

	if len(expectedGraph) != len(resultingGraph) {
		t.Error("Resulting graph of wrong size")
	}
	for i := range expectedGraph {
		if resultingGraph[i] != expectedGraph[i] {
			t.Errorf("Unexpected sitemap graph: %s != %s",
				resultingGraph[i], expectedGraph[i])
		}
	}

	if len(expectedWorkUnits) != len(resultingWorkUnits) {
		t.Error("Resulting work units of wrong size")
	}
	for i := range expectedWorkUnits {
		if expectedWorkUnits[i] != resultingWorkUnits[i] {
			t.Errorf("Unexpected work unit: %s != %s",
				expectedWorkUnits[i], resultingWorkUnits[i])
		}
	}
}

func TestParsePageRoutine(t *testing.T) {
	workChannel := make(chan string, 2)
	resultChannel := make(chan edge)

	workChannel <- testUrl

	go parsePageRoutine(workChannel, resultChannel)

	result := <-resultChannel

	expected := edge{
		source: testUrl,
		dest:   "/some-page",
	}

	if result != expected {
		t.Error("Result not expected")
	}
}
