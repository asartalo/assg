package e2e

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"codeberg.org/asartalo/assg/internal/commands"
	"github.com/stretchr/testify/assert"
)

func TestBasicSite(t *testing.T) {
	RunBuildTest("basic", t, false)
}

func TestSiteHomeOnly(t *testing.T) {
	RunBuildTest("site-home-only", t, false)
}

func TestStaticFiles(t *testing.T) {
	RunBuildTest("static-files", t, false)
}

func TestBlogExample(t *testing.T) {
	RunBuildTest("blog-posts", t, false)
}

func TestExtraData(t *testing.T) {
	RunBuildTest("extra-data", t, false)
}

func TestPreAndPostBuild(t *testing.T) {
	RunBuildTest("pre-and-post-build", t, false)
}

func TestFeeds(t *testing.T) {
	RunBuildTest("feeds", t, false)
}

func RunBuildTest(fixture string, t *testing.T, verbose bool) {
	t.Parallel()
	cwd, err := os.Getwd()
	fixtureDirectory := path.Join(cwd, "fixtures")
	assert.NoError(t, err, "Unable to get working directory")

	publicDir, err := os.MkdirTemp("", fmt.Sprintf("%s-public", fixture))
	assert.NoError(t, err, "Failed to create temp directory %s", publicDir)
	defer os.RemoveAll(publicDir)

	siteDir := path.Join(fixtureDirectory, fixture)
	expectedDir := path.Join(siteDir, "public-expected")
	now, err := time.Parse(time.RFC3339, "2024-03-01T10:00:00Z")
	assert.NoError(t, err)

	err = commands.Build(siteDir, publicDir, false, verbose, now)
	assert.NoError(t, err)

	assertDirContents(t, expectedDir, publicDir)
}
