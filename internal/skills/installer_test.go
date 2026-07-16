package skills

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// errReadDirFS wraps an fstest.MapFS and forces ReadDir to fail for a
// specific path, letting tests simulate a WalkDir error without a real
// broken filesystem.
type errReadDirFS struct {
	fstest.MapFS
	failPath string
}

func (e errReadDirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == e.failPath {
		return nil, errors.New("simulated read error")
	}
	return e.MapFS.ReadDir(name)
}

func skillMD(desc string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("---\ndescription: " + desc + "\n---\nbody\n")}
}

// TestList_FileCountExcludesRootDir is a regression test for the off-by-one
// in List(): WalkDir visits the skill's own root directory as well as its
// contents, so a naive count over every visited entry over-counts by one.
func TestList_FileCountExcludesRootDir(t *testing.T) {
	fsys := fstest.MapFS{
		".agents/skills/ibm-cloud-logs-foo/SKILL.md":       skillMD("does a thing"),
		".agents/skills/ibm-cloud-logs-foo/reference.md":   &fstest.MapFile{Data: []byte("ref")},
		".agents/skills/ibm-cloud-logs-foo/scripts/run.sh": &fstest.MapFile{Data: []byte("#!/bin/sh")},
	}

	inst := NewInstaller(fsys, "test")
	skills, err := inst.List()
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	// Entries under the skill root, NOT counting the root dir itself:
	// SKILL.md, reference.md, scripts/, scripts/run.sh = 4.
	want := 4
	if skills[0].FileCount != want {
		t.Errorf("FileCount = %d, want %d (root dir must not be counted)", skills[0].FileCount, want)
	}
}

// TestList_SurfacesWalkDirErrors is a regression test: List() used to
// silently discard fs.WalkDir errors (`_ = fs.WalkDir(...)`), so a broken
// embedded filesystem would report a bogus zero/partial FileCount instead of
// failing loudly.
func TestList_SurfacesWalkDirErrors(t *testing.T) {
	base := fstest.MapFS{
		".agents/skills/ibm-cloud-logs-foo/SKILL.md":       skillMD("does a thing"),
		".agents/skills/ibm-cloud-logs-foo/scripts/run.sh": &fstest.MapFile{Data: []byte("#!/bin/sh")},
	}
	fsys := errReadDirFS{MapFS: base, failPath: ".agents/skills/ibm-cloud-logs-foo/scripts"}

	inst := NewInstaller(fsys, "test")
	_, err := inst.List()
	if err == nil {
		t.Fatal("expected List() to surface the WalkDir error, got nil")
	}
}

// buildTestSkillFS returns an embedded-style FS with one skill containing a
// regular file and a script, for exercising Install()'s permission handling.
func buildTestSkillFS() fstest.MapFS {
	return fstest.MapFS{
		".agents/skills/ibm-cloud-logs-foo/SKILL.md":       skillMD("does a thing"),
		".agents/skills/ibm-cloud-logs-foo/reference.md":   &fstest.MapFile{Data: []byte("reference content")},
		".agents/skills/ibm-cloud-logs-foo/scripts/run.sh": &fstest.MapFile{Data: []byte("#!/bin/sh\necho hi\n")},
	}
}

// TestInstall_TightensPermissions verifies installed directories are 0700,
// regular files are 0600, and scripts (.sh/.py) are 0700 - private to the
// installing user, since skills live under the user's home directory.
func TestInstall_TightensPermissions(t *testing.T) {
	dest := t.TempDir()
	// Nest the actual install target so we can also assert the skill's own
	// subdirectories got tightened, not just files.
	target := filepath.Join(dest, "install-root")

	inst := NewInstaller(buildTestSkillFS(), "test")
	installed, err := inst.Install(target, false)
	if err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if len(installed) != 1 || installed[0] != "ibm-cloud-logs-foo" {
		t.Fatalf("unexpected installed skills: %v", installed)
	}

	skillDir := filepath.Join(target, "ibm-cloud-logs-foo")
	scriptsDir := filepath.Join(skillDir, "scripts")

	assertPerm(t, skillDir, 0700, true)
	assertPerm(t, scriptsDir, 0700, true)
	assertPerm(t, filepath.Join(skillDir, "SKILL.md"), 0600, false)
	assertPerm(t, filepath.Join(skillDir, "reference.md"), 0600, false)
	assertPerm(t, filepath.Join(scriptsDir, "run.sh"), 0700, false)
}

func assertPerm(t *testing.T, path string, want os.FileMode, wantDir bool) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.IsDir() != wantDir {
		t.Fatalf("%s: IsDir() = %v, want %v", path, info.IsDir(), wantDir)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s: perm = %o, want %o", path, got, want)
	}
}
