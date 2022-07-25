package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type parameter struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

// Scrapes the Elasticsearch Ingest Node docs to produce a JSON document
// containing the parameters for each processor.
func main() {
	var elasticsearchVersion = flag.String("es-version", "current", "Documentation branch (e.g. current, 8.4, etc)")
	flag.Parse()

	// Instantiate collector
	c := colly.NewCollector()
	c.CacheDir = ".cache"
	c.SetRequestTimeout(30 * time.Second)

	var processorTitle string
	processors := map[string][]parameter{}

	c.OnRequest(func(req *colly.Request) {
		log.Println("Requested:", req.URL.Path)
	})

	c.OnHTML("div.navfooter span.next a", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if strings.Contains(link, "processor.html") || link == "attachment.html" {
			e.Request.Visit(link)
		}
	})

	c.OnHTML("div.titlepage h2.title", func(e *colly.HTMLElement) {
		allText := e.Text
		childrenText := e.DOM.Children().Text()

		title := strings.TrimSuffix(allText, childrenText)

		title = strings.ToLower(title)
		title = strings.TrimSuffix(title, "processor")
		title = strings.TrimSpace(title)
		title = strings.ReplaceAll(title, " ", "_")
		processorTitle = title
	})

	c.OnHTML("div.table table", func(e *colly.HTMLElement) {
		summary, found := e.DOM.Attr("summary")
		if !found {
			return
		}

		if !strings.Contains(strings.ToLower(summary), "options") {
			log.Printf("Skipping table %q", summary)
			return
		}

		e.ForEach("tbody tr", func(_ int, e *colly.HTMLElement) {
			var p parameter

			e.ForEach("tr p", func(i int, element *colly.HTMLElement) {
				text := strings.ReplaceAll(element.Text, "\n", " ")
				switch i {
				case 0:
					p.Name = text
				case 1:
					switch text {
					case "yes", "yes*", "yes *":
						p.Required = true
					case "no":
						p.Required = false
					default:
						panic("unknown value " + text + " for " + processorTitle)
					}
				case 2:
					if text != "-" {
						p.Default = text
					}
				case 3:
					p.Description = text
				}
			})

			params, _ := processors[processorTitle]
			params = append(params, p)
			processors[processorTitle] = params
		})
	})

	err := c.Visit("https://www.elastic.co/guide/en/elasticsearch/reference/" + *elasticsearchVersion + "/processors.html")
	if err != nil {
		log.Fatal(err)
	}

	if len(processors) > 0 {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		enc.Encode(processors)
	}
}
