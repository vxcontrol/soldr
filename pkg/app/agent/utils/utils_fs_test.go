package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type getDirsToCreateTestCase struct {
	RootDir              string
	DstDir               string
	ExpectedDirsToCreate []string
}

func Test_getDirsToCreate(t *testing.T) {
	testCases := []getDirsToCreateTestCase{
		{
			RootDir: "some/dir",
			DstDir:  "some/dir/in/another/one",
			ExpectedDirsToCreate: []string{
				"some/dir/in/another/one",
				"some/dir/in/another",
				"some/dir/in",
			},
		},
		{
			RootDir:              "some/dir",
			DstDir:               "some/dir",
			ExpectedDirsToCreate: []string{},
		},
		{
			RootDir:              "some/dir",
			DstDir:               "some/dir/in",
			ExpectedDirsToCreate: []string{"some/dir/in"},
		},
	}

	for i, tc := range testCases {
		i, tc := i, tc
		t.Run(getTestCaseName(i), func(t *testing.T) {
			tmpDir, cleanupTmpDir, err := createTmpDir(t, "", "vxagent_test_getDirsToCreate_")
			if err != nil {
				t.Errorf("failed to create a tmp dir: %v", err)
				return
			}
			defer cleanupTmpDir()

			// test with an absolute path
			if err := testGetDirsToCreateWithRootDir(tmpDir, &tc); err != nil {
				t.Errorf("testing with an absolute path failed: %v", err)
				return
			}

			// test with a relative path
			oldWd, err := chwd(tmpDir)
			if err != nil {
				t.Error(err)
				return
			}
			defer func() {
				if err := os.Chdir(oldWd); err != nil {
					t.Errorf("failed to change back to the dir %s: %v", oldWd, err)
				}
			}()
			if err := testGetDirsToCreateWithRootDir(".", &tc); err != nil {
				t.Errorf("testing with a relative path failed: %v", err)
				return
			}
		})
	}
}

func testGetDirsToCreateWithRootDir(tmpDir string, tc *getDirsToCreateTestCase) error {
	expectedDirsToCreate := make([]string, len(tc.ExpectedDirsToCreate))
	for i := range tc.ExpectedDirsToCreate {
		expectedDirsToCreate[i] = filepath.Join(tmpDir, filepath.FromSlash(tc.ExpectedDirsToCreate[i]))
	}
	rootDir := filepath.Join(tmpDir, filepath.FromSlash(tc.RootDir))
	if err := os.MkdirAll(rootDir, 0o755|os.ModeSetgid); err != nil {
		return fmt.Errorf("failed to create a root dir %s: %w", rootDir, err)
	}
	dstDir := filepath.Join(tmpDir, filepath.Join(tc.DstDir))
	actualDirsToCreate, err := getDirsToCreate(dstDir)
	if err != nil {
		return fmt.Errorf("getDirsToCreate failed: %w", err)
	}
	if err := compareStringSlices(expectedDirsToCreate, actualDirsToCreate); err != nil {
		return fmt.Errorf("expected and actual dirs to create are different: %w", err)
	}
	return nil
}

type checkDstDirTestCase struct {
	RootDir                string
	DstDir                 string
	ExpectedDirsToExist    []string
	ExpectedDirsToNotExist []string
}

func Test_checkDstDir(t *testing.T) {
	testCases := []checkDstDirTestCase{
		{
			RootDir: "some/dir",
			DstDir:  "some/dir/in/another/one",
			ExpectedDirsToExist: []string{
				"some/dir",
				"some",
			},
			ExpectedDirsToNotExist: []string{
				"some/dir/in/another/one",
				"some/dir/in/another",
				"some/dir/in",
			},
		},
		{
			RootDir: "some/dir",
			DstDir:  "some/dir",
			ExpectedDirsToExist: []string{
				"some",
				"some/dir",
			},
			ExpectedDirsToNotExist: []string{},
		},
		{
			RootDir: "some/dir",
			DstDir:  "some/dir/in",
			ExpectedDirsToExist: []string{
				"some",
				"some/dir",
			},
			ExpectedDirsToNotExist: []string{
				"some/dir/in",
			},
		},
	}

	for i, tc := range testCases {
		i, tc := i, tc
		t.Run(getTestCaseName(i), func(t *testing.T) {
			tmpDir, cleanupTmpDir, err := createTmpDir(t, "", "vxagent_test_checkDstDir_")
			if err != nil {
				t.Errorf("failed to create a tmp dir: %v", err)
				return
			}
			defer cleanupTmpDir()

			// test with an absolute path
			if err := testCheckDstDir(tmpDir, &tc); err != nil {
				t.Errorf("testing with an absolute path failed: %v", err)
				return
			}

			// test with a relative path
			oldWd, err := chwd(tmpDir)
			if err != nil {
				t.Error(err)
				return
			}
			defer func() {
				if err := os.Chdir(oldWd); err != nil {
					t.Errorf("failed to change back to the dir %s: %v", oldWd, err)
				}
			}()
			if err := testCheckDstDir(".", &tc); err != nil {
				t.Errorf("testing with a relative path failed: %v", err)
				return
			}
		})
	}
}

func testCheckDstDir(tmpDir string, tc *checkDstDirTestCase) error {
	rootDir := filepath.Join(tmpDir, filepath.FromSlash(tc.RootDir))
	if err := os.MkdirAll(rootDir, 0o755|os.ModeSetgid); err != nil {
		return fmt.Errorf("failed to create the root dir %s: %w", rootDir, err)
	}
	dstDir := filepath.Join(tmpDir, filepath.FromSlash(tc.DstDir))
	expectedDirsToExist := make([]string, len(tc.ExpectedDirsToExist))
	for i, expected := range tc.ExpectedDirsToExist {
		expectedDirsToExist[i] = filepath.Join(tmpDir, filepath.FromSlash(expected))
	}
	expectedDirsToNotExist := make([]string, len(tc.ExpectedDirsToNotExist))
	for i, expected := range tc.ExpectedDirsToNotExist {
		expectedDirsToNotExist[i] = filepath.Join(tmpDir, filepath.FromSlash(expected))
	}

	cleanupFn, err := checkDstDir(dstDir)
	if err != nil {
		return fmt.Errorf("checkDstDir failed: %w", err)
	}

	type testErr error
	var someErr testErr = testErr(errors.New("some err"))

	err = cleanupFn(someErr)
	if !errors.Is(err, someErr) || err.Error() != someErr.Error() {
		return fmt.Errorf("expected error from the cleanup function: %v, got: %v", someErr, err)
	}

	for _, dirToExist := range expectedDirsToExist {
		if _, err := os.Stat(dirToExist); err != nil {
			return fmt.Errorf("failed to get the info of the dir %s expected to exist: %w", dirToExist, err)
		}
	}
	for _, dirToNotExist := range expectedDirsToNotExist {
		_, err := os.Stat(dirToNotExist)
		if err == nil {
			return fmt.Errorf("got the info of the dir %s expected to not exist: %w", dirToNotExist, err)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("while getting the dir %s info, got an unexpected error: %w", dirToNotExist, err)
		}
	}
	return nil
}

func getTestCaseName(i int) string {
	return fmt.Sprintf("test case #%d", i)
}

func createTmpDir(t *testing.T, dir string, pattern string) (string, func(), error) {
	tmpDir, err := ioutil.TempDir(dir, pattern)
	if err != nil {
		return "", nil, err
	}
	t.Logf("created a temp dir %s", tmpDir)
	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("tmpDir %s cleanup failed: %v", tmpDir, err)
		}
	}
	return tmpDir, cleanup, nil
}

func chwd(dst string) (string, error) {
	oldWd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get the current working dir: %w", err)
	}
	if err := os.Chdir(dst); err != nil {
		return "", fmt.Errorf("failed to change to the dir %s: %w", dst, err)
	}
	return oldWd, nil
}

func compareStringSlices(expected []string, actual []string) error {
	if len(expected) != len(actual) {
		return fmt.Errorf(
			"slices length is different: expected %v (length %d), got %v (length %d)",
			expected,
			len(expected),
			actual,
			len(actual),
		)
	}
	for i, e := range expected {
		a := actual[i]
		if a == e {
			continue
		}
		return fmt.Errorf("expected slice %v and actual slice %v have different elements with the index %d: %s and %s, respectively",
			expected,
			actual,
			i,
			e,
			a,
		)
	}
	return nil
}
