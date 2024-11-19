package e2e

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

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

func (suite *E2ETestSuite) TestStaticFiles() {
	// Static files are just copied over to the public directory
	suite.RunBuildTest("static-files")
}

func (suite *E2ETestSuite) TestBlogExample() {
	suite.RunBuildTest("blog-posts")
}

func (suite *E2ETestSuite) TestExtraData() {
	suite.RunBuildTest("extra-data")
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
	now, err := time.Parse(time.RFC3339, "2024-03-01T10:00:00Z")
	suite.NoError(err)

	err = commands.Build(siteDir, publicDir, false, false, now)
	suite.NoError(err)

	assertDirContents(suite.T(), expectedDir, publicDir)
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
