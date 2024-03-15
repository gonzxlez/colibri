# Colibri
Colibri is an extensible web crawling and scraping framework for Go, used to crawl and extract structured data on the web.

See [webextractor](webextractor/README.md).

## Installation
```
 $ go get github.com/gonzxlez/colibri
```

## Do
```go
// Do makes an HTTP request based on the rules.
func (c *Colibri) Do(rules *Rules) (resp Response, err error) 
```
```go
var rawRules = []byte(`{...}`) // Raw Rules ~ JSON 

c := colibri.New()
c.Client = ...    // Required
c.Delay = ...     // Optional
c.RobotsTxt = ... // Optional
c.Parser = ...    // Optional

var rules colibri.Rules
err := json.Unmarshal(rawRules, &rules)
if err != nil {
	panic(err)
} 

resp, err := c.Do(rules)
if err != nil {
	panic(err)
}

fmt.Println("URL:", resp.URL())
fmt.Println("Status code:", resp.StatusCode())
fmt.Println("Content-Type", resp.Header().Get("Content-Type"))
```

## Extract
```go
// Extract makes the HTTP request and parses the content of the response based on the rules.
func (c *Colibri) Extract(rules *Rules) (output *Output, err error)
```
```go
var rawRules = []byte(`{...}`) // Raw Rules ~ JSON 

c := colibri.New()
c.Client = ...    // Required
c.Delay = ...     // Optional
c.RobotsTxt = ... // Optional
c.Parser = ...    // Required

var rules colibri.Rules
err := json.Unmarshal(rawRules, &rules)
if err != nil {
	panic(err)
} 

output, err := c.Extract(&rules)
if err != nil {
	panic(err)
}

fmt.Println("URL:", output.Response.URL())
fmt.Println("Status code:", output.Response.StatusCode())
fmt.Println("Content-Type", output.Response.Header().Get("Content-Type"))
fmt.Println("Data:", output.Data)
```

# Raw  Rules ~ JSON
```json
{
	"Method": "string",
	"URL": "string",
	"Proxy": "string",
	"Header": {
		"string": "string",
		"string": ["string", "string", ...]
	},
	"Timeout": "number_millisecond",
	"Cookies": "bool",
	"IgnoreRobotsTxt": "bool",
	"Delay": "number_millisecond",
	"Redirects": "number",
	"Selectors": {...}
}
```

## Selectors
```json
{
	"Selectors": {
		"key_name": "expression"
	}
}
```
```json
{
	"Selectors": {
		"title": "//head/title"
	}
}
```

```json
{
	"Selectors": {
		"key_name":  {
			"Expr": "expression",
			"Type": "expression_type",
			"All": "bool",
			"Follow": "bool",
			"Method": "string",
			"Header": {...},
			"Proxy": "string",
			"Timeout": "number_millisecond",
			"Selectors": {...}
		}
	}
}
```
```json
{
	"Selectors": {
		"title":  {
			"Expr": "//head/title",
			"Type": "xpath"
		}
	}
}
```

### Nested selectors
```json
{
	"Selectors": {
		"body":  {
			"Expr": "//body",
			"Type": "xpath",
			"Selectors": {
				"p": "//p"
			}
		}
	}
}
```

### Find all
```json
{
	"Selectors": {
		"a":  {
			"Expr": "//body/a",
			"Type": "xpath",
			"All": true,
		}
	}
}
```

### Follow URLs
```json
{
	"Selectors": {
		"a":  {
			"Expr": "//body/a",
			"Type": "xpath",
			"All": true,
			"Follow": true,
			"Selectors": {
				"title": "//head/title"
			}
		}
	}
}
```

```json
{
	"Selectors": {
		"a":  {
			"Expr": "//body/a",
			"Type": "xpath",
			"All": true,
			"Follow": true,
			"Proxy": "http://proxy-url.com:8080",
			"Cookies": true,
			"Selectors": {
				"title": "//head/title"
			}
		}
	}
}
```

### Extra Fields
```json
{
	"Selectors": {
		"title":  {
			"Expr": "//head/title",
			"Type": "xpath",
			
			"Required": true
		}
	}
}
```

##  Example
```json
{
	"Method": "GET",
	"URL": "https://example.com",
	"Header": {
		"User-Agent": "test/0.1.0",
	},
	"Timeout": 5000,
	"Selectors": {
		"a":  {
			"Expr": "//body/a",
			"Type": "xpath",
			"All": true,
			"Follow": true,
			"Selectors": {
				"title": "//head/title"
			}
		}
	}
}
```