package examples_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerLabel(t *testing.T) {
	t.Parallel()
	wf := "jobs:\n  test:\n    runs-on: ubuntu-latest  # default\n    steps: []\n  mac:\n    runs-on: macos-14\n  lint:\n    runs-on: \"ubuntu-latest\"\n"
	dir := scratch(t, map[string]string{".github/workflows/ci.yml": wf})
	vars := runTwice(t, "runner-label", dir)
	want := "jobs:\n  test:\n    runs-on: ubuntu-24.04  # default\n    steps: []\n  mac:\n    runs-on: macos-14\n  lint:\n    runs-on: \"ubuntu-24.04\"\n"
	if got := read(t, filepath.Join(dir, ".github/workflows/ci.yml")); got != want {
		t.Errorf("workflow = %q", got)
	}
	if vars["jobs"] != "test, lint" {
		t.Errorf("jobs = %v", vars["jobs"])
	}
}

func TestRenovatePreset(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"renovate.json": "{\n    \"extends\": [\"config:base\"],\n    \"schedule\": [\"weekly\"]\n}\n"})
	runTwice(t, "renovate-preset", dir)
	if got := read(t, filepath.Join(dir, "renovate.json")); got != "{\n    \"extends\": [\"github>acme/renovate-config\"],\n    \"schedule\": [\"weekly\"]\n}\n" {
		t.Errorf("renovate.json = %q", got)
	}
}

func TestRuffConfig(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{"pyproject.toml": "[project]\nname = \"api\"\n\n[tool.ruff]\nline-length = 88  # was black's\n"})
	runTwice(t, "ruff-config", dir)
	if got := read(t, filepath.Join(dir, "pyproject.toml")); got != "[project]\nname = \"api\"\n\n[tool.ruff]\nline-length = 88  # was black's\n\n[tool.ruff.lint]\nextend-select = [\"E\", \"F\", \"I\"]\n" {
		t.Errorf("pyproject.toml = %q", got)
	}
	runScript(t, "ruff-config", t.TempDir())
}

func TestGradleWrapperBump(t *testing.T) {
	t.Parallel()
	props := "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5-all.zip\ndistributionSha256Sum=c16d517b50dd28b3f5838f0e844b7520b8f1eb610f2f29de7e4e04a1b7c9c79b\n"
	dir := scratch(t, map[string]string{"gradle/wrapper/gradle-wrapper.properties": props})
	runTwice(t, "gradle-wrapper-bump", dir)
	got := read(t, filepath.Join(dir, "gradle/wrapper/gradle-wrapper.properties"))
	if !strings.Contains(got, "gradle-8.10.2-all.zip") || !strings.Contains(got, "distributionSha256Sum=2ab88d6de2c23e6adae7363ae6e29cbdd2a709e992929b48b6530fd0c7133bd6") {
		t.Errorf("properties = %q", got)
	}
}

func TestJVMDependencyBump(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"pom.xml":                   "<project>\n  <properties>\n    <jackson.version>2.15.2</jackson.version>\n  </properties>\n  <dependencies>\n    <dependency>\n      <groupId>com.fasterxml.jackson.core</groupId>\n      <artifactId>jackson-databind</artifactId>\n      <version>${jackson.version}</version>\n    </dependency>\n  </dependencies>\n</project>\n",
		"gradle/libs.versions.toml": "[libraries]\njackson = \"com.fasterxml.jackson.core:jackson-databind:2.15.2\"\n",
	})
	vars := runTwice(t, "jvm-dependency-bump", dir)
	if got := read(t, filepath.Join(dir, "pom.xml")); !strings.Contains(got, "<jackson.version>2.17.2</jackson.version>") {
		t.Errorf("pom.xml = %q", got)
	}
	if got := read(t, filepath.Join(dir, "gradle/libs.versions.toml")); !strings.Contains(got, "jackson-databind:2.17.2") {
		t.Errorf("catalog = %q", got)
	}
	if vars["files"] != "pom.xml, gradle/libs.versions.toml" {
		t.Errorf("files = %v", vars["files"])
	}
}

func TestDotnet8(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"src/Api/Api.csproj":       "<Project Sdk=\"Microsoft.NET.Sdk.Web\">\n  <PropertyGroup>\n    <TargetFramework>net6.0</TargetFramework>\n  </PropertyGroup>\n</Project>\n",
		"test/Api.Tests.csproj":    "<Project>\n  <ItemGroup>\n    <PackageReference Include=\"Microsoft.AspNetCore.Mvc.Testing\" Version=\"6.0.25\" />\n  </ItemGroup>\n</Project>\n",
		"global.json":              "{\n  \"sdk\": {\n    \"version\": \"6.0.400\"\n  }\n}\n",
		"src/Legacy/Legacy.csproj": "<Project>\n  <PropertyGroup>\n    <TargetFramework>net48</TargetFramework>\n  </PropertyGroup>\n</Project>\n",
	})
	runTwice(t, "dotnet-8", dir)
	for path, want := range map[string]string{
		"src/Api/Api.csproj":       "<TargetFramework>net8.0</TargetFramework>",
		"test/Api.Tests.csproj":    `Version="8.0.8"`,
		"global.json":              `"version": "8.0.400"`,
		"src/Legacy/Legacy.csproj": "<TargetFramework>net48</TargetFramework>",
	} {
		if got := read(t, filepath.Join(dir, path)); !strings.Contains(got, want) {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestHelmImageTag(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"charts/api/Chart.yaml":    "apiVersion: v2\nname: api\nversion: 1.4.9\nappVersion: \"2.3.1\"\n",
		"charts/api/values.yaml":   "image:\n  repository: ghcr.io/acme/api\n  tag: \"2.3.1\"\n",
		"charts/other/Chart.yaml":  "apiVersion: v2\nname: other\nversion: 0.1.0\n",
		"charts/other/values.yaml": "image:\n  repository: nginx\n  tag: \"1.27\"\n",
	})
	runTwice(t, "helm-image-tag", dir)
	if got := read(t, filepath.Join(dir, "charts/api/Chart.yaml")); got != "apiVersion: v2\nname: api\nversion: 1.4.10\nappVersion: \"2.4.0\"\n" {
		t.Errorf("Chart.yaml = %q", got)
	}
	if got := read(t, filepath.Join(dir, "charts/api/values.yaml")); !strings.Contains(got, `tag: "2.4.0"`) {
		t.Errorf("values.yaml = %q", got)
	}
	if got := read(t, filepath.Join(dir, "charts/other/Chart.yaml")); !strings.Contains(got, "version: 0.1.0") {
		t.Errorf("other chart bumped: %q", got)
	}
}
