package internal_test

import (
	"net/http"
	"testing"

	"github.com/buildpacks/libcnb/mocks"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/internal"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testGetEolDate(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks.ExitHandler
	)

	it.Before(func() {
		httpmock.Activate()
		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)
	})

	it.After(func() {
		httpmock.DeactivateAndReset()
	})

	context("finds release cycle by major.minor version", func() {
		it.Before(func() {
			httpmock.RegisterResponder(http.MethodGet, "https://endoflife.date/api/foo.json", httpmock.NewBytesResponder(200, []byte(`
[
	{
		"cycle": "10.1",
		"releaseDate": "2022-09-23",
		"eol": false,
		"minJavaVersion": 11,
		"latest": "10.1.24",
		"latestReleaseDate": "2024-05-09",
		"lts": false
	},
	{
		"cycle": "10.0",
		"releaseDate": "2020-12-03",
		"eol": "2026-12-31",
		"minJavaVersion": 8,
		"latest": "10.0.27",
		"latestReleaseDate": "2025-10-03",
		"lts": false
	},
	{
		"cycle": "9",
		"releaseDate": "2017-09-27",
		"eol": "2023-12-31",
		"minJavaVersion": 8,
		"latest": "9.0.89",
		"latestReleaseDate": "2021-05-03",
		"lts": false
	}
]`)))
		})

		it("finds release cycle by major and minor version", func() {
			eolDate, err := internal.GetEolDate("foo", "10.0.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(eolDate).To(Equal("2026-12-31T00:00:00Z"))
		})

		it("finds release cycle by major version", func() {
			eolDate, err := internal.GetEolDate("foo", "9.5.4")
			Expect(err).NotTo(HaveOccurred())
			Expect(eolDate).To(Equal("2023-12-31T00:00:00Z"))
		})

		it("returns empty eol date if eol is a bool", func() {
			eolDate, err := internal.GetEolDate("foo", "10.1.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(eolDate).To(Equal(""))
		})

	})

}
