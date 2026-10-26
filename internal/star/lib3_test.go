package star

import (
	"strings"
	"testing"
)

func TestLibYAMLPath(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"set and add": {
			repo: map[string]string{
				"a.yaml": "# top\nname: api   # the name\n\"quoted key\": 'x'\nlist:\n  - name: inner\n    value: 1\nblock: |\n  name: not a key\n  more\nmeta:\n    labels:\n        app: api\n---\nname: second\n",
			},
			script: `load("@fleet//yamlpath.star", "get", "set", "values")
def change(repo):
    set(repo, "a.yaml", "name", "web")
    set(repo, "a.yaml", "quoted key", "it's")
    set(repo, "a.yaml", "meta.labels.tier", "1.20")
    set(repo, "a.yaml", "meta.annotations.team", "payments")
    set(repo, "a.yaml", "new.deep.flag", True)
    set(repo, "a.yaml", "tags", ["a", "b c", None, 3])
    repo.set_var("block", get(repo, "a.yaml", "block") == None)
    repo.set_var("inner", get(repo, "a.yaml", "list.name") == None)
    repo.set_var("missing", get(repo, "nope.yaml", "x") == None)
    repo.set_var("star", repr(values(repo, "a.yaml", "meta.*.app")))
    set(repo, "new.yaml", "a.b", "yes")
`,
			want: map[string]string{
				"a.yaml":   "# top\nname: web   # the name\n\"quoted key\": 'it''s'\nlist:\n  - name: inner\n    value: 1\nblock: |\n  name: not a key\n  more\nmeta:\n    labels:\n        app: api\n        tier: \"1.20\"\n    annotations:\n        team: payments\nnew:\n  deep:\n    flag: true\ntags: [a, b c, null, 3]\n---\nname: second\n",
				"new.yaml": "a:\n  b: \"yes\"\n",
			},
			vars: map[string]string{"block": "true", "inner": "true", "missing": "true", "star": `{"meta.labels.app": "api"}`},
		},
		"wildcards": {
			repo: map[string]string{".github/workflows/ci.yml": "jobs:\n  a:\n    runs-on: ubuntu-latest\n  b:\n    runs-on: \"ubuntu-latest\"\n  c:\n    steps: []\n"},
			script: `load("@fleet//yamlpath.star", "delete", "set")
def change(repo):
    p = ".github/workflows/ci.yml"
    repo.set_var("set", set(repo, p, "jobs.*.runs-on", "ubuntu-24.04"))
    repo.set_var("none", set(repo, p, "jobs.*.timeout", 5))
    delete(repo, p, "jobs.c")
    repo.set_var("gone", delete(repo, p, "jobs.zzz"))
    repo.set_var("nofile", delete(repo, "nope.yml", "a"))
`,
			want: map[string]string{".github/workflows/ci.yml": "jobs:\n  a:\n    runs-on: ubuntu-24.04\n  b:\n    runs-on: \"ubuntu-24.04\"\n"},
			vars: map[string]string{"set": "true", "none": "false", "gone": "false", "nofile": "false"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibJSONAndTOML(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"jsonfile": {
			repo: map[string]string{"a.json": "{\n\t\"a\": 1\n}\n"},
			script: `load("@fleet//jsonfile.star", "delete", "get", "read", "set", "write")
def change(repo):
    set(repo, "new/b.json", "x.y", [1, 2])
    set(repo, "a.json", "a", 2)
    repo.set_var("same", set(repo, "a.json", "a", 2))
    repo.set_var("unchanged", write(repo, "a.json", read(repo, "a.json")))
    repo.set_var("def", get(repo, "a.json", "a.b.c", default = "none"))
    repo.set_var("nofile", get(repo, "zz.json", "a") == None)
    repo.set_var("d1", delete(repo, "a.json", "zz"))
    repo.set_var("d2", delete(repo, "a.json", "x.y"))
    repo.set_var("d3", delete(repo, "zz.json", "a"))
    repo.set_var("dflt", read(repo, "zz.json", default = []))
`,
			want: map[string]string{"a.json": "{\n\t\"a\": 2\n}\n", "new/b.json": "{\n  \"x\": {\n    \"y\": [\n      1,\n      2\n    ]\n  }\n}\n"},
			vars: map[string]string{"same": "false", "unchanged": "false", "def": "none", "nofile": "true", "d1": "false", "d2": "false", "d3": "false", "dflt": "[]"},
		},
		"toml": {
			repo: map[string]string{
				"pyproject.toml": "# config\nname = 'api'\n\n[project]\ndependencies = [\n  \"a\",  # first\n  \"b\",\n]\n\n[[tool.mypy.overrides]]\nmodule = \"x\"\n\n[tool.empty]\n",
			},
			script: `load("@fleet//toml.star", "delete", "get", "set")
def change(repo):
    p = "pyproject.toml"
    repo.set_var("deps", get(repo, p, "project", "dependencies"))
    repo.set_var("name", get(repo, p, "", "name"))
    repo.set_var("override", get(repo, p, "tool.mypy.overrides", "module") == None)
    set(repo, p, "", "name", "web")
    set(repo, p, "", "version", "1.0")
    set(repo, p, "project", "dependencies", ["c"])
    set(repo, p, "tool.empty", "on", True)
    set(repo, p, "tool.new", "opts", {"a": 1, "b": "x"})
    repo.set_var("none", get(repo, "zz.toml", "", "a") == None)
    repo.set_var("gone", delete(repo, p, "nope", "x"))
    repo.set_var("nofile", delete(repo, "zz.toml", "", "x"))
    set(repo, "fresh.toml", "", "a", 1.5)
`,
			want: map[string]string{
				"pyproject.toml": "# config\nname = 'web'\nversion = \"1.0\"\n\n[project]\ndependencies = [\"c\"]\n\n[[tool.mypy.overrides]]\nmodule = \"x\"\n\n[tool.empty]\non = true\n\n[tool.new]\nopts = { a = 1, b = \"x\" }\n",
				"fresh.toml":     "a = 1.5\n",
			},
			vars: map[string]string{"deps": "[\n  \"a\",  # first\n  \"b\",\n]", "name": "api", "override": "true", "none": "true", "gone": "false", "nofile": "false"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibJVMAndDotnet(t *testing.T) {
	t.Parallel()
	sha := strings.Repeat("ab", 32)
	for name, tc := range map[string]libCase{
		"gradle": {
			repo: map[string]string{
				"build.gradle":              "dependencies {\n  implementation 'org.slf4j:slf4j-api:2.0.7'\n  implementation \"org.slf4j:slf4j-api:$slf4jVersion\"\n  implementation 'org.slf4j:slf4j-api-extra:1.0'\n}\n",
				"gradle/libs.versions.toml": "[libraries]\nslf4j = { module = \"org.slf4j:slf4j-api\", version = \"2.0.7\" }\nslf4j2 = { group = \"org.slf4j\", name = \"slf4j-api\", version = \"2.0.7\" }\nslf4j3 = \"org.slf4j:slf4j-api:2.0.7\"\n",
				WrapperPath:                 "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.5-all.zip\ndistributionSha256Sum=old\n",
			},
			script: `load("@fleet//gradle.star", "set_dependency", "set_wrapper_version")
def change(repo):
    repo.set_var("changed", set_dependency(repo, "org.slf4j:slf4j-api", "2.0.16"))
    set_wrapper_version(repo, "8.10.2", sha256 = {"bin": "wrong", "all": "` + sha + `"})
    repo.set_var("lower", set_wrapper_version(repo, "8.1"))
`,
			want: map[string]string{
				"build.gradle":              "dependencies {\n  implementation 'org.slf4j:slf4j-api:2.0.16'\n  implementation \"org.slf4j:slf4j-api:$slf4jVersion\"\n  implementation 'org.slf4j:slf4j-api-extra:1.0'\n}\n",
				"gradle/libs.versions.toml": "[libraries]\nslf4j = { module = \"org.slf4j:slf4j-api\", version = \"2.0.16\" }\nslf4j2 = { group = \"org.slf4j\", name = \"slf4j-api\", version = \"2.0.16\" }\nslf4j3 = \"org.slf4j:slf4j-api:2.0.16\"\n",
				WrapperPath:                 "distributionUrl=https\\://services.gradle.org/distributions/gradle-8.10.2-all.zip\ndistributionSha256Sum=" + sha + "\n",
			},
			vars: map[string]string{"changed": "[build.gradle gradle/libs.versions.toml]", "lower": "false"},
		},
		"gradle no wrapper": {
			script: `load("@fleet//gradle.star", "set_wrapper_version", "wrapper_version")
def change(repo):
    repo.set_var("set", set_wrapper_version(repo, "8.10"))
    repo.set_var("v", wrapper_version(repo) == None)
`,
			vars: map[string]string{"set": "false", "v": "true"},
		},
		"maven": {
			repo: map[string]string{"pom.xml": `<project>
  <build>
    <pluginManagement>
      <plugins>
        <plugin>
          <artifactId>maven-surefire-plugin</artifactId>
          <version>3.0.0</version>
        </plugin>
        <plugin>
          <groupId>org.other</groupId>
          <artifactId>maven-surefire-plugin</artifactId>
          <version>1.0</version>
        </plugin>
      </plugins>
    </pluginManagement>
  </build>
  <dependencies>
    <dependency>
      <groupId>org.slf4j</groupId>
      <artifactId>slf4j-api</artifactId>
    </dependency>
  </dependencies>
</project>
`},
			script: `load("@fleet//maven.star", "set_dependency")
def change(repo):
    set_dependency(repo, "org.apache.maven.plugins", "maven-surefire-plugin", "3.5.0")
    repo.set_var("noversion", set_dependency(repo, "org.slf4j", "slf4j-api", "2.0.16"))
`,
			want: map[string]string{"pom.xml": "<project>\n  <build>\n    <pluginManagement>\n      <plugins>\n        <plugin>\n          <artifactId>maven-surefire-plugin</artifactId>\n          <version>3.5.0</version>\n        </plugin>\n        <plugin>\n          <groupId>org.other</groupId>\n          <artifactId>maven-surefire-plugin</artifactId>\n          <version>1.0</version>\n        </plugin>\n      </plugins>\n    </pluginManagement>\n  </build>\n  <dependencies>\n    <dependency>\n      <groupId>org.slf4j</groupId>\n      <artifactId>slf4j-api</artifactId>\n    </dependency>\n  </dependencies>\n</project>\n"},
			vars: map[string]string{"noversion": "[]"},
		},
		"dotnet": {
			repo: map[string]string{
				"Directory.Packages.props": "<Project>\n  <ItemGroup>\n    <PackageVersion Include=\"Serilog\" Version=\"3.0.1\" />\n    <PackageVersion Update=\"Polly\" Version=\"7.0\" />\n  </ItemGroup>\n</Project>\n",
				"src/A/A.csproj":           "<Project>\n  <PropertyGroup>\n    <TargetFrameworks>net6.0;net48</TargetFrameworks>\n  </PropertyGroup>\n  <ItemGroup>\n    <PackageReference Include=\"Serilog\" />\n  </ItemGroup>\n</Project>\n",
			},
			script: `load("@fleet//dotnet.star", "set_package", "set_sdk_version", "set_target_framework")
def change(repo):
    repo.set_var("pkgs", set_package(repo, "serilog", "4.0.0"))
    set_package(repo, "Polly", "8.4.0")
    repo.set_var("tfm", set_target_framework(repo, "net8.0"))
    repo.set_var("sdk", set_sdk_version(repo, "8.0.400"))
`,
			want: map[string]string{"Directory.Packages.props": "<Project>\n  <ItemGroup>\n    <PackageVersion Include=\"Serilog\" Version=\"4.0.0\" />\n    <PackageVersion Update=\"Polly\" Version=\"8.4.0\" />\n  </ItemGroup>\n</Project>\n"},
			vars: map[string]string{"pkgs": "[Directory.Packages.props]", "tfm": "[]", "sdk": "false"},
		},
		"helm": {
			repo: map[string]string{
				"Chart.yaml":  "name: api\ndependencies:\n  - repository: https://x\n    name: redis\n    version: \"17.0.0\"\n  - name: other\n    version: 1.0.0\nnested:\n  dependencies:\n    - name: redis\n      version: 1.0.0\n",
				"values.yaml": "image:\n  tag: v1  # pinned\n  repository: \"envoyproxy/envoy\"\nother:\n  repository: envoyproxy/envoy\n",
			},
			script: `load("@fleet//helm.star", "charts", "set_dependency", "set_image_tag")
def change(repo):
    repo.set_var("charts", charts(repo))
    set_dependency(repo, "redis", "19.x")
    set_image_tag(repo, "envoyproxy/envoy", "v1.31.0")
`,
			want: map[string]string{
				"Chart.yaml":  "name: api\ndependencies:\n  - repository: https://x\n    name: redis\n    version: \"19.x\"\n  - name: other\n    version: 1.0.0\nnested:\n  dependencies:\n    - name: redis\n      version: 1.0.0\n",
				"values.yaml": "image:\n  tag: v1.31.0  # pinned\n  repository: \"envoyproxy/envoy\"\nother:\n  repository: envoyproxy/envoy\n",
			},
			vars: map[string]string{"charts": "[.]"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibBatchErrors(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		script string
		repo   map[string]string
		want   string
	}{
		"yaml nested":  {"load('@fleet//yamlpath.star', 'set')\ndef change(repo):\n  set(repo, 'a.yaml', 'a', 1)\n", map[string]string{"a.yaml": "a:\n  b: 1\n"}, "a in a.yaml holds a nested value"},
		"yaml scalar":  {"load('@fleet//yamlpath.star', 'set')\ndef change(repo):\n  set(repo, 'a.yaml', 'a.b', 1)\n", map[string]string{"a.yaml": "a: 1\n"}, "a in a.yaml is a scalar"},
		"yaml dict":    {"load('@fleet//yamlpath.star', 'set')\ndef change(repo):\n  set(repo, 'a.yaml', 'a', {})\n", nil, "can't write a dict"},
		"json missing": {"load('@fleet//jsonfile.star', 'read')\ndef change(repo):\n  read(repo, 'a.json')\n", nil, "a.json does not exist"},
		"json scalar":  {"load('@fleet//jsonfile.star', 'set')\ndef change(repo):\n  set(repo, 'a.json', 'a.b', 1)\n", map[string]string{"a.json": "{\"a\": 1}"}, "a in a.json is not an object"},
		"toml nested":  {"load('@fleet//toml.star', 'set')\ndef change(repo):\n  set(repo, 'a.toml', '', 'a', [[1]])\n", nil, "can't write a list here"},
		"gradle sha":   {"load('@fleet//gradle.star', 'set_wrapper_version')\ndef change(repo):\n  set_wrapper_version(repo, '8.10')\n", map[string]string{WrapperPath: "distributionUrl=https\\://x/gradle-8.5-bin.zip\ndistributionSha256Sum=x\n"}, "pins distributionSha256Sum; pass sha256"},
	} {
		s, err := load(t, map[string]string{"change.star": tt.script}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		dir := files(t, tt.repo)
		if _, err := s.Run(t.Context(), dir, RepoInfo{}, nil, 0); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

// WrapperPath is where gradle.star reads the wrapper properties.
const WrapperPath = "gradle/wrapper/gradle-wrapper.properties"
