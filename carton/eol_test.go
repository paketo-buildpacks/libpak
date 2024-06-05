package carton_test

import (
	"net/http"
	"testing"

	"github.com/buildpacks/libcnb/mocks"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/carton"
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
		"eol": "2022-10-31",
		"minJavaVersion": 8,
		"latest": "10.0.27",
		"latestReleaseDate": "2022-10-03",
		"lts": false
	},
	{
		"cycle": "9",
		"releaseDate": "2017-09-27",
		"eol": false,
		"minJavaVersion": 8,
		"latest": "9.0.89",
		"latestReleaseDate": "2024-05-03",
		"lts": false
	}
]`)))
		})

		it("returns correct cycle", func() {
			eolDate, err := carton.GetEolDate("foo", "10.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(eolDate).To(Equal("2022-10-31T00:00:00Z"))
		})
	})

	context("finds release cycle by major version", func() {
		it.Before(func() {
			httpmock.RegisterResponder(http.MethodGet, "https://endoflife.date/api/foo.json", httpmock.NewBytesResponder(200, []byte(`
[
	{
		"cycle": "10",
		"releaseDate": "2022-09-23",
		"eol": "2024-10-31",
		"minJavaVersion": 11,
		"latest": "10.1.24",
		"latestReleaseDate": "2024-05-09",
		"lts": false
	},
	{
		"cycle": "9",
		"releaseDate": "2020-12-03",
		"eol": "2022-10-31",
		"minJavaVersion": 8,
		"latest": "9.0.27",
		"latestReleaseDate": "2022-10-03",
		"lts": false
	}
]`)))
		})

		it("returns correct cycle", func() {
			eolDate, err := carton.GetEolDate("foo", "10.1.1")
			Expect(err).NotTo(HaveOccurred())
			Expect(eolDate).To(Equal("2024-10-31T00:00:00Z"))
		})
	})

	context("cycle.EOL is bool", func() {
		it.Before(func() {
			httpmock.RegisterResponder(http.MethodGet, "https://endoflife.date/api/foo.json", httpmock.NewBytesResponder(200, []byte(`
[
	{
		"cycle": "10",
		"releaseDate": "2022-09-23",
		"eol": false,
		"minJavaVersion": 11,
		"latest": "10.1.24",
		"latestReleaseDate": "2024-05-09",
		"lts": false
	},
	{
		"cycle": "9",
		"releaseDate": "2020-12-03",
		"eol": "2022-10-31",
		"minJavaVersion": 8,
		"latest": "9.0.27",
		"latestReleaseDate": "2022-10-03",
		"lts": false
	}
]`)))
		})

		it("returns empty eol date", func() {
			eolDate, err := carton.GetEolDate("foo", "10.0.0")
			Expect(err).NotTo(HaveOccurred())
			Expect(eolDate).To(Equal(""))
		})
	})
}
