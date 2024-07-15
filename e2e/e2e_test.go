package e2e

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/asartalo/assg/internal/commands"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
	FixtureDirectory string
}

func (suite *E2ETestSuite) TestBasicSite() {
	suite.RunBuildTest("basic")
}

func (suite *E2ETestSuite) TestSiteHomeOnly() {
	suite.RunBuildTest("site-home-only")
}

func (suite *E2ETestSuite) SetupSuite() {
	// current working directory
	cwd, err := os.Getwd()
	suite.NoError(err, "Unable to get working directory")
	suite.FixtureDirectory = path.Join(cwd, "fixtures")
}

func (suite *E2ETestSuite) RunBuildTest(fixture string) {
	// Setup test environment
	publicDir, err := os.MkdirTemp("", fmt.Sprintf("%s-public", fixture))
	suite.NoError(err, "Failed to create temp directory %s", publicDir)
	defer os.RemoveAll(publicDir)

	siteDir := path.Join(suite.FixtureDirectory, fixture)
	expectedDir := path.Join(siteDir, "public-expected")

	err = commands.Build(siteDir, publicDir, false)
	suite.NoError(err)

	assertDirContents(suite.T(), expectedDir, publicDir)
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
