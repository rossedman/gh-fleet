package campaign

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Limits on what Fingerprint hashes: past them, a campaign directory is
// more likely a repo or a home directory than a campaign.
const (
	maxFingerprintFiles = 2000
	maxFingerprintBytes = 50 << 20
)

// errNoFingerprint stops a walk that can't produce a fingerprint.
var errNoFingerprint = errors.New("no fingerprint")

// remoteRef finds the ref of a remote Starlark load, as in
// load("github.com/acme/lib/docker.star@v1.2.0", ...).
var remoteRef = regexp.MustCompile(`load\(\s*["'][^"'@]+/[^"'@]*@([^"'/]+)`)

// pinnedRef matches a version tag or a commit SHA, which name the same
// code every time, unlike a branch.
var pinnedRef = regexp.MustCompile(`^(v?\d+(\.\d+)*|[0-9a-f]{7,64})$`)

// Fingerprint hashes everything the change can read on the campaign's
// side: every file under its directory (hidden directories aside), and any
// script or local Starlark load outside it. Two runs with the same
// fingerprint on the same commit of a repo make the same change. It
// returns "" when that can't be told: the directory is too big to hash, or
// a Starlark file loads a remote module at a branch.
func (c *Campaign) Fingerprint() string {
	h := sha256.New()
	seen := map[string]bool{}
	files, size := 0, int64(0)
	add := func(p string) error {
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		if seen[abs] {
			return nil
		}
		seen[abs] = true
		b, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		files++
		size += int64(len(b))
		if files > maxFingerprintFiles || size > maxFingerprintBytes {
			return errNoFingerprint
		}
		if strings.HasSuffix(abs, ".star") {
			for _, m := range remoteRef.FindAllSubmatch(b, -1) {
				if !pinnedRef.Match(m[1]) {
					return errNoFingerprint
				}
			}
		}
		write(h, abs, b)
		return nil
	}
	dir := c.Dir()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return add(p)
	})
	if err != nil {
		return ""
	}
	// Files outside the directory: scripts, and Starlark loads followed
	// from each step's file.
	var outside []string
	for _, s := range c.Change.Steps {
		if p := c.ScriptFile(s); p != "" {
			outside = append(outside, p)
		}
		if p := c.StarlarkFile(s); p != "" {
			outside = append(outside, starlarkFiles(p, map[string]bool{})...)
		}
	}
	sort.Strings(outside)
	for _, p := range outside {
		if err := add(p); err != nil {
			return ""
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// write adds one file to h, framed by its path and length so different
// files can't hash alike.
func write(h hash.Hash, path string, b []byte) {
	fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(path), len(b))
	h.Write(b)
}

// starlarkFiles returns p and every local file it loads, recursively.
func starlarkFiles(p string, seen map[string]bool) []string {
	if seen[p] {
		return nil
	}
	seen[p] = true
	out := []string{p}
	b, err := os.ReadFile(p)
	if err != nil {
		return out // hashing p reports the error
	}
	for _, m := range localLoad.FindAllStringSubmatch(string(b), -1) {
		out = append(out, starlarkFiles(filepath.Join(filepath.Dir(p), m[1]), seen)...)
	}
	return out
}
