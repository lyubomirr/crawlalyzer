package crawl

import (
	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
)

type Crawler interface {
	Crawl(urls []string, onResp func(response Response), onQuit func())
}

type crawledLink struct {
	currentLink *url.URL
	startURL *url.URL
}

type concurrentCrawler struct {
	semaphore chan struct{}
	visited map[url.URL]struct{}
	visitedMux sync.Mutex
	pendingLinks chan crawledLink
	followExternalLinks bool
	ctx context.Context
	cancel context.CancelFunc
}

type Response struct {
	body *html.Node
	StartURL *url.URL
	URL *url.URL
	HTML string
	Headers http.Header
	Cookies []*http.Cookie
	ScriptLinks []string
	Scripts []string
	Styles []string
}

func NewConcurrentCrawler(maxGoroutines int, followExternalLinks bool) Crawler {
	ctx, cancel := context.WithCancel(context.Background())
	return &concurrentCrawler {
		semaphore: make(chan struct{}, maxGoroutines),
		visited: make(map[url.URL]struct{}),
		pendingLinks: make(chan crawledLink),
		followExternalLinks: followExternalLinks,
		ctx: ctx,
		cancel: cancel,
	}
}

func (c* concurrentCrawler) Crawl(urls []string, onResp func(response Response), onQuit func()) {
	go func() {
		os.Stdin.Read(make([]byte, 1))
		c.cancel()
	}()

	go func() {
		for _, u := range urls {
			parsed, err := url.Parse(u)
			if err == nil {
				c.pendingLinks <- crawledLink{currentLink: parsed, startURL: parsed}
			}
		}
	}()

	for {
		select {
		case l := <- c.pendingLinks:
			go c.crawlLink(l, onResp)
		case <- c.ctx.Done():
			c.cancel()
			go func() {
				for range c.pendingLinks {
					<-c.pendingLinks
				}
			}()
			onQuit()
			return
		}
	}
}

func (c *concurrentCrawler) crawlLink(link crawledLink, onResp func(response Response)) {
	c.semaphore <- struct{}{}
	r, err := parse(link)
	if err != nil {
		//Just ignore this link.
		return
	}

	onResp(r)
	links := c.getLinks(nil, r.body, r.URL, r.StartURL)
	for _, li := range links {
		c.pendingLinks <- crawledLink{currentLink: li, startURL: link.startURL}
	}
	<- c.semaphore
}

func parse(c crawledLink) (Response, error) {
	r, err := http.Get(c.currentLink.String())
	if err != nil {
		return Response{}, err
	}
	defer r.Body.Close()

	b, err := html.Parse(r.Body)
	if err != nil {
		return Response{}, err
	}
	scriptLinks, scripts, styles := getResources(nil, nil, nil, b, c.currentLink)

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return Response{}, err
	}

	resp := Response{
		StartURL: c.startURL,
		URL: c.currentLink,
		body:    b,
		Headers: r.Header,
		Cookies: r.Cookies(),
		HTML: string(bytes),
		ScriptLinks: scriptLinks,
		Scripts: scripts,
		Styles: styles,
	}

	return resp, nil
}

func getResources(
	scriptLinks []string, scripts []string, styles []string, n *html.Node, baseURL *url.URL) ([]string, []string, []string) {
	if n == nil {
		return scriptLinks, scripts, styles
	}

	if n.Data == "script" {
		for _, a := range n.Attr {
			if a.Key == "src" {
				script, err := readContent(a.Val, baseURL)
				if err != nil {
					break //just ignore the url
				}
				scriptLinks = append(scriptLinks, a.Val)
				scripts = append(scripts, script)
			}
		}
	}

	if n.Type == html.ElementNode && n.Data == "link" {
		isStyle := false
		for _, a := range n.Attr {
			if a.Key == "rel" && a.Val == "stylesheet" {
				isStyle = true
			}
		}
		if isStyle {
			for _, a := range n.Attr {
				if a.Key == "href" {
					style, err := readContent(a.Val, baseURL)
					if err != nil {
						break
					}
					styles = append(styles, style)
				}
			}
		}
	}

	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		scriptLinks, scripts, styles = getResources(scriptLinks, scripts, styles, ch, baseURL)
	}

	return scriptLinks, scripts, styles
}

func readContent(link string, baseURL *url.URL) (string, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return "", nil
	}
	if parsed.Host == "" {
		//Relative url -> add the base url
		parsed = baseURL.ResolveReference(parsed)
	}
	resp, err := http.Get(parsed.String())
	if err != nil {
		return "", err
	}
	
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (c *concurrentCrawler) getLinks(
	links []*url.URL, n *html.Node, baseURL *url.URL, startURL *url.URL) []*url.URL {
	if n == nil {
		return links
	}

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				link, err := url.Parse(a.Val)
				if err != nil {
					break
				}
				if link.Host == "" {
					//Relative url -> add the base url
					link = baseURL.ResolveReference(link)
				}
				if c.followExternalLinks || link.Host == startURL.Host {
					links = c.addIfUnvisited(link, links)
				}
			}
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		links = c.getLinks(links, ch, baseURL, startURL)
	}
	return links
}

func (c *concurrentCrawler) addIfUnvisited(l *url.URL, links []*url.URL) []*url.URL {
	c.visitedMux.Lock()
	defer c.visitedMux.Unlock()

	if !c.IsVisited(*l) {
		c.visited[*l] = struct{}{}
		return append(links, l)
	}

	return links
}

func (c *concurrentCrawler) IsVisited(url url.URL) bool {
	 _, ok := c.visited[url]
	return ok
}


