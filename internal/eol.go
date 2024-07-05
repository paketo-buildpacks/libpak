package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Masterminds/semver/v3"
)

const eolBaseURL = "https://endoflife.date/api"

func GetEolDate(eolID, version string) (string, error) {
	cycleList, err := getProjectCycleList(eolID)
	if err != nil {
		return "", fmt.Errorf("could not fetch cycle list: %w", err)
	}

	cycle, err := selectCycle(version, cycleList)
	if err != nil {
		return "", fmt.Errorf("could not find a release cycle: %w", err)
	}

	if cycle.EOL == "" {
		return "", nil
	}

	eol, err := time.Parse(time.DateOnly, cycle.EOL)
	if err != nil {
		return "", fmt.Errorf("could not parse eol %q: %w", cycle.EOL, err)
	}

	return eol.Format(time.RFC3339), nil
}

func selectCycle(version string, cycles cycleList) (*cycle, error) {
	versionParsed, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}

	for _, v := range []string{fmt.Sprintf("%d.%d", versionParsed.Major(), versionParsed.Minor()), fmt.Sprintf("%d", versionParsed.Major())} {
		for _, c := range cycles {
			if c.Cycle == v {
				return c, nil
			}
		}
	}

	return nil, fmt.Errorf("no release cycle found for the version %s", version)
}

func getProjectCycleList(id string) (cycleList, error) {
	res, err := http.Get(fmt.Sprintf("%s/%s.json", eolBaseURL, id))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch release cycles, status: %d", res.StatusCode)
	}

	cycles := cycleList{}
	if err := json.NewDecoder(res.Body).Decode(&cycles); err != nil {
		return nil, err
	}

	return cycles, nil
}

type cycleList []*cycle

type cycle struct {
	Cycle string
	EOL   string
}

func (c *cycle) UnmarshalJSON(data []byte) error {
	var i map[string]interface{}

	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}

	if val, ok := i["cycle"]; ok {
		switch t := val.(type) {
		case string:
			c.Cycle = t
		case int, float64:
			c.Cycle = fmt.Sprintf("%d", t)
		default:
			return fmt.Errorf("invalid type of the \"cycle\" field: %T", t)
		}
	}

	if val, ok := i["eol"]; ok {
		switch t := val.(type) {
		case string:
			c.EOL = t
		case bool:
			c.EOL = ""
		default:
			return fmt.Errorf("invalid type of the \"eol\" field: %T", t)
		}
	}

	return nil
}
