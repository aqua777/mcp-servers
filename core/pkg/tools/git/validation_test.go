package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type validationTestSuite struct {
	suite.Suite
	tmpDir string
}

func (s *validationTestSuite) SetupTest() {
	dir, err := os.MkdirTemp("", "git-validation-test-*")
	s.Require().NoError(err)
	s.tmpDir = dir
}

func (s *validationTestSuite) TearDownTest() {
	os.RemoveAll(s.tmpDir)
}

// validateRepoPath tests

func (s *validationTestSuite) TestValidateRepoPath_NoRestriction() {
	// No allowed repository set — any path is permitted
	err := validateRepoPath("/any/arbitrary/path", "")
	s.NoError(err)
}

func (s *validationTestSuite) TestValidateRepoPath_ExactMatch() {
	allowed := filepath.Join(s.tmpDir, "repo")
	s.Require().NoError(os.Mkdir(allowed, 0o755))

	err := validateRepoPath(allowed, allowed)
	s.NoError(err)
}

func (s *validationTestSuite) TestValidateRepoPath_Subdirectory() {
	allowed := filepath.Join(s.tmpDir, "repo")
	subdir := filepath.Join(allowed, "subdir")
	s.Require().NoError(os.MkdirAll(subdir, 0o755))

	err := validateRepoPath(subdir, allowed)
	s.NoError(err)
}

func (s *validationTestSuite) TestValidateRepoPath_OutsideAllowed() {
	allowed := filepath.Join(s.tmpDir, "allowed")
	outside := filepath.Join(s.tmpDir, "outside")
	s.Require().NoError(os.Mkdir(allowed, 0o755))
	s.Require().NoError(os.Mkdir(outside, 0o755))

	err := validateRepoPath(outside, allowed)
	s.Error(err)
	s.Contains(err.Error(), "outside the allowed repository")
}

func (s *validationTestSuite) TestValidateRepoPath_TraversalAttempt() {
	allowed := filepath.Join(s.tmpDir, "allowed")
	s.Require().NoError(os.Mkdir(allowed, 0o755))

	// allowed/../outside — after EvalSymlinks this resolves outside of allowed
	outside := filepath.Join(s.tmpDir, "outside")
	s.Require().NoError(os.Mkdir(outside, 0o755))

	traversal := filepath.Join(allowed, "..", "outside")
	err := validateRepoPath(traversal, allowed)
	s.Error(err)
	s.Contains(err.Error(), "outside the allowed repository")
}

func (s *validationTestSuite) TestValidateRepoPath_SymlinkEscape() {
	allowed := filepath.Join(s.tmpDir, "allowed")
	outside := filepath.Join(s.tmpDir, "outside")
	s.Require().NoError(os.Mkdir(allowed, 0o755))
	s.Require().NoError(os.Mkdir(outside, 0o755))

	// Create a symlink inside allowed that points to outside
	symlink := filepath.Join(allowed, "escape_link")
	s.Require().NoError(os.Symlink(outside, symlink))

	err := validateRepoPath(symlink, allowed)
	s.Error(err)
	s.Contains(err.Error(), "outside the allowed repository")
}

func (s *validationTestSuite) TestValidateRepoPath_NonExistentRepoPath() {
	// When repoPath doesn't exist, EvalSymlinks fails — should return an error
	allowed := filepath.Join(s.tmpDir, "allowed")
	s.Require().NoError(os.Mkdir(allowed, 0o755))

	nonExistent := filepath.Join(s.tmpDir, "does_not_exist")
	err := validateRepoPath(nonExistent, allowed)
	s.Error(err)
}

func (s *validationTestSuite) TestValidateRepoPath_NonExistentAllowedPath() {
	// When allowedRepository doesn't exist, EvalSymlinks fails — should return an error
	existing := filepath.Join(s.tmpDir, "existing")
	s.Require().NoError(os.Mkdir(existing, 0o755))

	nonExistentAllowed := filepath.Join(s.tmpDir, "nonexistent_allowed")
	err := validateRepoPath(existing, nonExistentAllowed)
	s.Error(err)
	s.Contains(err.Error(), "invalid allowed repository path")
}

// validateRefName tests

func (s *validationTestSuite) TestValidateRefName_Valid() {
	validNames := []string{
		"main",
		"feature/my-branch",
		"HEAD~1",
		"abc123def",
		"v1.0.0",
		"refs/heads/main",
	}
	for _, name := range validNames {
		err := validateRefName(name)
		s.NoError(err, "expected valid ref name: %q", name)
	}
}

func (s *validationTestSuite) TestValidateRefName_FlagInjection() {
	injectionAttempts := []string{
		"-p",
		"--help",
		"--output=/tmp/evil",
		"--orphan=evil",
		"-f",
	}
	for _, name := range injectionAttempts {
		err := validateRefName(name)
		s.Error(err, "expected rejection of ref name: %q", name)
		s.Contains(err.Error(), "cannot start with '-'")
	}
}

func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(validationTestSuite))
}
