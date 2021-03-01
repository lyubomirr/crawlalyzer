package main

import (
	"crawler/analyzer"
	"crawler/crawl"
	"fmt"
	"log"
)

const maxConcurrentGoroutimes = 10

//Usage: go run main.go -urls=https://google.com -follow-external=true
func main() {
	urls, followExternalLinks := getArgs()

	webAnalyzer, err := analyzer.NewWebAnalyzer()
	if err != nil {
		log.Fatal(err)
	}

	crawler := crawl.NewConcurrentCrawler(maxConcurrentGoroutimes, followExternalLinks)
	fmt.Println("Starting to crawl...Press enter to stop crawling.")
	go loading()

	crawler.Crawl(urls, func(response crawl.Response) {
		webAnalyzer.Analyze(response)
	}, func() {
		webAnalyzer.SaveResult()
		fmt.Printf("Results saved in '%v'.\n", analyzer.ResultsFileName)
	})
}
