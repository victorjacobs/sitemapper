package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type edge struct {
	source string
	dest   string
}

var linkRegex *regexp.Regexp

func init() {
	linkRegex, _ = regexp.Compile("href=\"([\\w/-]+)\"")
}

func main() {
	var err error

	baseUrl, outputPath, err := parseFlags()
	if err != nil {
		panic(err.Error())
	}

	workChannel, edgeChannel, resultChannel := startSiteMapBuilderCoordinator(baseUrl, 5 * time.Second)

	for i := 0; i < 20; i++ {
		go parsePageRoutine(workChannel, edgeChannel)
	}

	edges := <-resultChannel

	fmt.Println()
	fmt.Println()
	fmt.Printf("Number of edges found: %d\n", len(edges))

	err = writeGraphToDotFile(edges, outputPath)
	if err != nil {
		panic(err.Error())
	}
}

// Parses command line flags, returns error if not enough arguments were passed.
func parseFlags() (baseUrl string, outputPath string, err error) {
	flag.Parse()

	if len(flag.Args()) != 2 {
		return "", "", errors.New("invalid number of arguments")
	}

	baseUrl = flag.Args()[0]
	outputPath = flag.Args()[1]

	return
}

// Starts a coordinator goroutine to build site map for given URL. Returns channels to attach workers to.
func startSiteMapBuilderCoordinator(
	baseUrl string,
	timeout time.Duration,
) (workChannel chan string, edgeChannel chan edge, resultChannel chan []edge) {
	workChannel = make(chan string, 1000)
	edgeChannel = make(chan edge)
	resultChannel = make(chan []edge)

	go func() {
		fmt.Println("Mapping " + baseUrl)

		pages := make(map[string]struct{})
		pages["/"] = struct{}{}
		siteMap := make([]edge, 0)

		workChannel <- baseUrl + "/"

		for {
			select {
			case e := <-edgeChannel:
				siteMap = append(siteMap, edge{
					source: strings.Replace(e.source, baseUrl, "", -1),
					dest:   e.dest,
				})

				if _, contains := pages[e.dest]; !contains {
					pages[e.dest] = struct{}{}

					workChannel <- baseUrl + e.dest

					fmt.Print(".")
					if len(pages)%50 == 0 {
						fmt.Printf(" %d\n", len(pages))
					}
				}
			case <-time.After(timeout):
				close(workChannel)
				resultChannel <- siteMap
				return
			}
		}
	}()

	return
}

// Parses web page at url coming in over workChannel and puts all discovered (relative) links into resultChannel.
func parsePageRoutine(workChannel <-chan string, resultChannel chan<- edge) {
	for url := range workChannel {
		for _, page := range getAllLinksInUrl(url) {
			resultChannel <- edge{
				source: url,
				dest:   page,
			}
		}
	}
}

// Downloads given URL and returns all links found in the document.
func getAllLinksInUrl(url string) []string {
	matches := make([]string, 0)

	urlContents, err := getUrlContents(url)
	if err != nil {
		fmt.Printf("WARNING: failed to get links from %s because %s", url, err.Error())
		return matches
	}

	for _, linkMatch := range linkRegex.FindAllStringSubmatch(urlContents, -1) {
		matches = append(matches, linkMatch[1])
	}

	return matches
}

// Downloads document at given URL and returns document content.
func getUrlContents(url string) (string, error) {
	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)

	return buf.String(), nil
}

// Generates Graphviz Dot definition of given site graph.
func generateGraphDotDefinition(edges []edge) string {
	var sb strings.Builder

	sb.WriteString("digraph Sitemap {\n")

	for _, edge := range edges {
		sb.WriteString(fmt.Sprintf("\t\"%s\" -> \"%s\";\n", edge.source, edge.dest))
	}

	sb.WriteString("}")

	return sb.String()
}

// Writes given edge list to dot file.
func writeGraphToDotFile(edges []edge, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	w := bufio.NewWriter(f)

	_, _ = w.WriteString(generateGraphDotDefinition(edges))

	_ = w.Flush()

	fmt.Printf("Wrote %d edges to %s\n", len(edges), path)

	return nil
}
