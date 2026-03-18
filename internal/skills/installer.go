// Package skills provides installation and management of embedded agent skills.
// Skills follow the agentskills.io open standard and work with Claude Code,
// Cursor, Gemini CLI, GitHub Copilot, and 30+ other AI agents.
package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// skillsRoot is the path prefix inside the embedded filesystem.
	skillsRoot = ".agents/skills"
	// skillPrefix is the naming prefix for all IBM Cloud Logs skills.
	skillPrefix = "ibm-cloud-logs"
)

// Installer manages agent skill installation from an embedded filesystem.
type Installer struct {
	fs      fs.FS
	version string
}

// NewInstaller creates a new skill installer.
// The provided filesystem should contain skills under .agents/skills/.
func NewInstaller(embeddedFS fs.FS, version string) *Installer {
	return &Installer{fs: embeddedFS, version: version}
}

// SkillInfo contains metadata about an available skill.
type SkillInfo struct {
	Name        string
	Description string
	Path        string
	FileCount   int
}

// List returns information about all available embedded skills.
func (inst *Installer) List() ([]SkillInfo, error) {
	entries, err := fs.ReadDir(inst.fs, skillsRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded skills: %w", err)
	}

	var skills []SkillInfo
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), skillPrefix) {
			continue
		}

		info := SkillInfo{
			Name: entry.Name(),
			Path: filepath.Join(skillsRoot, entry.Name()),
		}

		// Extract description from SKILL.md frontmatter
		skillMD, err := fs.ReadFile(inst.fs, filepath.Join(info.Path, "SKILL.md"))
		if err == nil {
			info.Description = extractDescription(string(skillMD))
		}

		// Count files in the skill
		count := 0
		_ = fs.WalkDir(inst.fs, info.Path, func(_ string, _ fs.DirEntry, _ error) error {
			count++
			return nil
		})
		info.FileCount = count

		skills = append(skills, info)
	}

	return skills, nil
}

// Install copies embedded skills to the destination directory.
// If dest is empty, it defaults to ~/.agents/skills/.
// Set projectLevel to true to install to ./.agents/skills/ instead.
func (inst *Installer) Install(dest string, projectLevel bool) ([]string, error) {
	if dest == "" {
		if projectLevel {
			dest = filepath.Join(".", ".agents", "skills")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to determine home directory: %w", err)
			}
			dest = filepath.Join(home, ".agents", "skills")
		}
	}

	skills, err := inst.List()
	if err != nil {
		return nil, err
	}

	var installed []string
	for _, skill := range skills {
		targetDir := filepath.Join(dest, skill.Name)

		// Remove old version if exists
		if err := os.RemoveAll(targetDir); err != nil {
			return installed, fmt.Errorf("failed to remove old skill %s: %w", skill.Name, err)
		}

		// Copy skill files from embedded FS to target
		if err := copyEmbeddedDir(inst.fs, skill.Path, targetDir); err != nil {
			return installed, fmt.Errorf("failed to install skill %s: %w", skill.Name, err)
		}

		installed = append(installed, skill.Name)
	}

	return installed, nil
}

// Remove removes installed IBM Cloud Logs skills from the destination directory.
func (inst *Installer) Remove(dest string, projectLevel bool) ([]string, error) {
	if dest == "" {
		if projectLevel {
			dest = filepath.Join(".", ".agents", "skills")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to determine home directory: %w", err)
			}
			dest = filepath.Join(home, ".agents", "skills")
		}
	}

	entries, err := os.ReadDir(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read directory %s: %w", dest, err)
	}

	var removed []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), skillPrefix) {
			target := filepath.Join(dest, entry.Name())
			if err := os.RemoveAll(target); err != nil {
				return removed, fmt.Errorf("failed to remove %s: %w", entry.Name(), err)
			}
			removed = append(removed, entry.Name())
		}
	}

	return removed, nil
}

// copyEmbeddedDir recursively copies a directory from an embedded FS to disk.
func copyEmbeddedDir(srcFS fs.FS, srcPath, destPath string) error {
	return fs.WalkDir(srcFS, srcPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path from source root
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destPath, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0750)
		}

		// Read from embedded FS
		data, err := fs.ReadFile(srcFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Determine file permissions
		perm := os.FileMode(0644)
		if strings.HasSuffix(path, ".sh") || strings.HasSuffix(path, ".py") {
			perm = 0755
		}

		// Write to disk
		return os.WriteFile(targetPath, data, perm)
	})
}

// extractDescription pulls the description from SKILL.md YAML frontmatter.
func extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	inDescription := false
	var descParts []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				break // end of frontmatter
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}

		if strings.HasPrefix(trimmed, "description:") {
			inDescription = true
			// Check for inline value after "description:"
			after := strings.TrimPrefix(trimmed, "description:")
			after = strings.TrimSpace(after)
			after = strings.TrimPrefix(after, ">")
			after = strings.TrimSpace(after)
			if after != "" {
				descParts = append(descParts, after)
			}
			continue
		}

		if inDescription {
			// Continuation lines start with spaces
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				descParts = append(descParts, trimmed)
			} else {
				break // next key
			}
		}
	}

	desc := strings.Join(descParts, " ")
	// Truncate for display
	if len(desc) > 120 {
		desc = desc[:117] + "..."
	}
	return desc
}
