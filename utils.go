package main

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

func getArgs() (urls []string, followExternalLinks bool) {
	urlsArg := flag.String("urls", "https://google.com", "Comma-separated urls to crawl.")
	followLinksArg := flag.Bool("follow-external", true, "Specifies whether to follow external links.")
	flag.Parse()

	urls = strings.Split(*urlsArg, ",")
	for _, url := range urls {
		url = strings.TrimSpace(url)
	}
	followExternalLinks = *followLinksArg
	return
}

func loading() {
	for {
		for i := 0; i <= 4; i++ {
			fmt.Printf("\rCrawling%v", strings.Repeat(".", i))
			time.Sleep(500 * time.Millisecond)
		}
	}
}