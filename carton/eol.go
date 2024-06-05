package carton

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
	eol "github.com/kobayashi/eol/pkg/api"
)

func GetEolDate(eolID, version string) (string, error) {
	cycleList, err := eol.NewHTTPClient().GetProjectCycleList(context.Background(), eolID)
	if err != nil {
		return "", fmt.Errorf("could not fetch cycle list: %w", err)
	}

	cycle, err := selectCycle(version, cycleList)
	if err != nil {
		return "", fmt.Errorf("could not find a relese cycle: %w", err)
	}

	if cycle.EOL.B != nil {
		return "", nil
	}

	eol, err := time.Parse(time.DateOnly, *cycle.EOL.S)
	if err != nil {
		return "", fmt.Errorf("could not parse eol %q: %w", *cycle.EOL.S, err)
	}

	return eol.Format(time.RFC3339), nil
}

func selectCycle(version string, cycles eol.CycleList) (*eol.Cycle, error) {
	versionParsed, err := semver.NewVersion(version)
	if err != nil {
		return nil, err
	}

	for _, v := range []string{fmt.Sprintf("%d.%d", versionParsed.Major(), versionParsed.Minor()), fmt.Sprintf("%d", versionParsed.Major())} {
		for _, c := range cycles {
			if (c.Cycle.S != nil && *c.Cycle.S == v) || (c.Cycle.I != nil && fmt.Sprintf("%d", *c.Cycle.I) == v) {
				return c, nil
			}
		}
	}

	return nil, fmt.Errorf("no release cycle found for the version %s", version)
}
