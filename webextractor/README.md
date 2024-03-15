# Colibri ~ WebExtractor
WebExtractor are default interfaces for Colibri ready to start crawling or extracting data on the web.

## Quick Starts

### Do
```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/gonzxlez/colibri"
	"github.com/gonzxlez/colibri/webextractor"
)

var rawRules = `{
	"Method": "GET",
	"URL": "https://example.com"
}`

func main() {
	we, err := webextractor.New()
	if err != nil {
		panic(err)
	}

	var rules colibri.Rules
	err = json.Unmarshal([]byte(rawRules), &rules)
	if err != nil {
		panic(err)
	}

	resp, err := we.Do(&rules)
	if err != nil {
		panic(err)
	}

	fmt.Println("URL:", resp.URL())
	fmt.Println("Status code:", resp.StatusCode())
	fmt.Println("Content-Type", resp.Header().Get("Content-Type"))
}
```
```
URL: https://example.com
Status code: 200
Content-Type text/html; charset=UTF-8
```

### Extract
```go
package main

import (
	"fmt"

	"github.com/gonzxlez/colibri"
	"github.com/gonzxlez/colibri/webextractor"
)

var rawRules = `{
	"Method": "GET",
	"URL":    "https://example.com",
	"Selectors": {
		"title": "//head/title"
	}
}`

func main() {
	we, err := webextractor.New()
	if err != nil {
		panic(err)
	}

	var rules colibri.Rules
	err = json.Unmarshal([]byte(rawRules), &rules)
	if err != nil {
		panic(err)
	}

	output, err := we.Extract(&rules)
	if err != nil {
		panic(err)
	}

	fmt.Println("URL:", output.Response.URL())
	fmt.Println("Status code:", output.Response.StatusCode())
	fmt.Println("Content-Type", output.Response.Header().Get("Content-Type"))
	fmt.Println("Data:", output.Data)
}

```
```
URL: https://example.com
Status code: 200
Content-Type text/html; charset=UTF-8
Data: map[title:Example Domain]
```