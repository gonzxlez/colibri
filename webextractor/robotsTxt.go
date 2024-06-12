package webextractor

import (
	"io"
	"net/url"
	"sync"

	"github.com/gonzxlez/colibri"

	"github.com/temoto/robotstxt"
)

const robotsTxtPath = "/robots.txt"

// RobotsData gets, stores and parses robots.txt restrictions.
type RobotsData struct {
	rw   sync.RWMutex
	data map[string]*robotstxt.RobotsData
}

// NewRobotsData returns a new RobotsData structure.
func NewRobotsData() *RobotsData {
	return &RobotsData{data: make(map[string]*robotstxt.RobotsData)}
}

// IsAllowed verifies that the User-Agent can access the URL.
// Gets and stores the robots.txt restrictions of the URL host and for use in URLs with the same host.
func (robots *RobotsData) IsAllowed(c *colibri.Colibri, rules *colibri.Rules) error {
	if rules.URL.Path == robotsTxtPath {
		return nil
	}

	robots.rw.RLock()
	robotsData, ok := robots.data[rules.URL.Host]
	robots.rw.RUnlock()

	if !ok {
		robotsRef, err := url.Parse(robotsTxtPath)
		if err != nil {
			return err
		}

		robotsRules := rules.Clone()

		robotsRules.Method = "GET"
		robotsRules.URL = rules.URL.ResolveReference(robotsRef)
		robotsRules.IgnoreRobotsTxt = true

		resp, err := c.Do(robotsRules)
		if err != nil {
			return err
		}

		buf, err := io.ReadAll(resp.Body())
		if err != nil {
			return err
		}

		robotsData, err = robotstxt.FromStatusAndBytes(resp.StatusCode(), buf)
		if err != nil {
			return err
		}

		robots.rw.Lock()
		robots.data[rules.URL.Host] = robotsData
		robots.rw.Unlock()

		colibri.ReleaseRules(robotsRules)
	}

	if robotsData.TestAgent(rules.URL.Path, rules.Header.Get("User-Agent")) {
		return nil
	}
	return colibri.ErrRobotstxtRestriction
}

// Clear removes stored robots.txt restrictions.
func (robots *RobotsData) Clear() {
	robots.rw.Lock()
	clear(robots.data)
	robots.rw.Unlock()
}
