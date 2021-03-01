# Concurrect web crawler which tries to detect used technologies

The crawler uses Wappalyzer's [technology fingerprints](https://github.com/AliasIO/wappalyzer/blob/master/src/technologies.json) in order to detect used technologies of a website.
After finishing all the technologies are aggregated per single root URL.

## Usage:
There are two command line arguments:
- urls - Comma-separated urls to crawl.
- follow-external - Specifies whether to follow external links.

Example usage:
`go run main.go -urls=https://google.com -follow-external=true`

When you press a key the crawling stops and the aggregated technologies for the root urls and their links are saved into `fingerprints.json` file.
