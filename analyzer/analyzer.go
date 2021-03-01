package analyzer

import (
	"crawler/crawl"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
)

const dataFileName = "technologies.json"
const ResultsFileName = "fingerprints.json"

type Analyzer interface {
	Analyze(response crawl.Response)
	SaveResult()
}

type Result struct {
	DetectedTechnologies map[DetectedTechnology]struct{}
	Implies              map[string]struct{}
	Excludes             map[string]struct{}
	Categories           map[string]struct{}
}

func (r Result) toMarshalableResult() marshalabeResult {
	res := marshalabeResult{
		DetectedTechnologies: make([]DetectedTechnology, 0, len(r.DetectedTechnologies)),
		Implies:              make([]string, 0, len(r.Implies)),
		Excludes:             make([]string, 0, len(r.Excludes)),
		Categories:           make([]string, 0, len(r.Categories)),
	}

	for k, _ := range r.DetectedTechnologies {
		res.DetectedTechnologies = append(res.DetectedTechnologies, k)
	}
	for k, _ := range r.Implies {
		res.Implies = append(res.Implies, k)
	}
	for k, _ := range r.Excludes {
		res.Excludes = append(res.Excludes, k)
	}
	for k, _ := range r.Categories {
		res.Categories = append(res.Categories, k)
	}

	return res
}

func getSetValues(set map[interface{}]struct{}) interface{} {
	values := make([]interface{}, 0, len(set))
	for k, _ := range set {
		values = append(values, k)
	}
	return values
}


type DetectedTechnology struct {
	Name        string
	Description string
	Website     string
}

func NewWebAnalyzer() (Analyzer, error) {
	analyzer := &webAnalyzer{result: make(map[string]Result)}
	data, err := loadData()
	if err != nil {
		return nil, fmt.Errorf("coulnd't create web analyzer: %w", err)
	}

	analyzer.data = data
	return analyzer, nil
}

type webAnalyzer struct {
	data      technologyPrints
	result    map[string]Result
	resultMux sync.Mutex
}

func loadData() (technologyPrints, error) {
	f, err := os.Open(dataFileName)
	if err != nil {
		return technologyPrints{}, fmt.Errorf("coulnd't open the file: %w", err)
	}

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return technologyPrints{}, fmt.Errorf("coulnd't read the file: %w", err)
	}

	prints := technologyPrints{}
	err = json.Unmarshal(bytes, &prints)
	if err != nil {
		return technologyPrints{}, fmt.Errorf("coulnd't unmarshal the file to JSON: %w", err)
	}

	err = f.Close()
	if err != nil {
		return technologyPrints{}, fmt.Errorf("couldn't close the file: %w", err)
	}

	return prints, nil
}

func (a *webAnalyzer) Analyze(response crawl.Response) {
	res := a.analyzeSingle(response)
	a.addToFinalResult(response.StartURL, res)
}

func (a *webAnalyzer) addToFinalResult(rootURL *url.URL, res Result) {
	urlString := rootURL.String()

	a.resultMux.Lock()
	_, ok := a.result[rootURL.String()]
	if !ok {
		a.result[urlString] = Result{
			DetectedTechnologies: make(map[DetectedTechnology]struct{}),
			Categories:           make(map[string]struct{}),
			Implies:              make(map[string]struct{}),
			Excludes:             make(map[string]struct{}),
		}
	}

	for t, _ := range res.DetectedTechnologies {
		a.result[urlString].DetectedTechnologies[t] = struct{}{}
	}
	for c, _ := range res.Categories {
		a.result[urlString].Categories[c] = struct{}{}
	}
	for i, _ := range res.Implies {
		a.result[urlString].Implies[i] = struct{}{}
	}
	for e, _ := range res.Excludes {
		a.result[urlString].Excludes[e] = struct{}{}
	}
	a.resultMux.Unlock()
}

func (a *webAnalyzer) analyzeSingle(response crawl.Response) Result {
	res := Result{
		DetectedTechnologies: make(map[DetectedTechnology]struct{}),
		Categories:           make(map[string]struct{}),
		Implies:              make(map[string]struct{}),
		Excludes:             make(map[string]struct{}),
	}

	for name, t := range a.data.Technologies {
		headersMatch := matchHeaders(t.Headers, response.Headers)
		if headersMatch {
			a.addToResult(name, t, &res)
			continue
		}

		cookiesMatch := matchCookies(t.Cookies, response.Cookies)
		if cookiesMatch {
			a.addToResult(name, t, &res)
			continue
		}

		scriptLinksMatch := matchSlice(t.Scripts, response.ScriptLinks)
		if scriptLinksMatch {
			a.addToResult(name, t, &res)
			continue
		}

		jsMatch := matchSlice(getValues(t.JS), response.Scripts)
		if jsMatch {
			a.addToResult(name, t, &res)
			continue
		}

		htmlMatch := matchSingle(t.HTML, response.HTML)
		if htmlMatch {
			a.addToResult(name, t, &res)
			continue
		}

		cssMatch := matchSlice(t.CSS, response.Styles)
		if cssMatch {
			a.addToResult(name, t, &res)
			continue
		}
	}
	return res
}

func getValues(m map[string]string) []string {
	values := make([]string, 0, len(m))
	for _, v := range m {
		if v != "" {
			values = append(values, v)
		}
	}
	return values
}

func matchHeaders(toMatch map[string]string, h http.Header) bool {
	for k, v := range toMatch {
		values := h.Values(k)
		if len(values) > 0 {
			//We just match header name
			if v == "" {
				return true
			}
			//Match header values
			re, err := regexp.Compile(v)
			if err != nil {
				//Regex with lookahead/behind are not supported and returns an error so I just skip them..
				//https://stackoverflow.com/questions/24836885/go-regex-error-parsing-regexp-invalid-escape-sequence-k
				continue
			}

			for _, val := range values {
				res := re.FindStringSubmatch(val)
				if len(res) > 0 {
					return true
				}
			}
		}
	}
	return false
}

func (a *webAnalyzer) addToResult(name string, t technology, res *Result) {
	tech := DetectedTechnology{
		Name:        name,
		Description: t.Description,
		Website:     t.Website,
	}
	res.DetectedTechnologies[tech] = struct{}{}

	for _, c := range t.Cats {
		catName := a.data.Categories[strconv.Itoa(c)]
		res.Categories[catName.Name] = struct{}{}
	}

	for _, i := range t.Implies {
		res.Implies[i] = struct{}{}
	}

	for _, e := range t.Excludes {
		res.Excludes[e] = struct{}{}
	}
}

func matchCookies(toMatch map[string]string, c []*http.Cookie) bool {
	for _, cookie := range c {
		val, ok := toMatch[cookie.Name]
		if !ok {
			continue
		}

		if val == "" {
			//No check for value
			return true
		}

		re, err := regexp.Compile(val)
		if err != nil {
			//Regex with lookahead/behind are not supported and returns an error so I just skip them..
			//https://stackoverflow.com/questions/24836885/go-regex-error-parsing-regexp-invalid-escape-sequence-k
			continue
		}

		res := re.FindStringSubmatch(cookie.Name)
		if len(res) > 0 {
			return true
		}
	}
	return false
}

func matchSlice(toMatch []string, values []string) bool {
	for _, match := range toMatch {
		re, err := regexp.Compile(match)
		if err != nil {
			//Regex with lookahead/behind are not supported and returns an error so I just skip them..
			//https://stackoverflow.com/questions/24836885/go-regex-error-parsing-regexp-invalid-escape-sequence-k
			continue
		}

		for _, val := range values {
			res := re.FindStringSubmatch(val)
			if len(res) > 0 {
				return true
			}
		}
	}
	return false
}

func matchSingle(toMatch []string, value string) bool {
	return matchSlice(toMatch, []string{value})
}

func (a *webAnalyzer) SaveResult() {
	toMarshal := make(map[string]marshalabeResult)
	for k, v := range a.result {
		toMarshal[k] = v.toMarshalableResult()
	}

	json, err := json.MarshalIndent(toMarshal, "", "    ")
	if err != nil {
		log.Fatalf("could'nt parse results to json: %v", err)
	}

	fmt.Println(string(json))
	err = ioutil.WriteFile(ResultsFileName, json, 0777)
	if err != nil {
		log.Fatalf("couldn't save results to file: %v", err)
	}
}
