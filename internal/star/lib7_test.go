package star

import (
	"strings"
	"testing"
)

func TestLibBatchB(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"bazel": {
			repo: map[string]string{
				"MODULE.bazel": "bazel_dep(version = \"1.0\", name = \"rules_x\", dev_dependency = True)\nbazel_dep(name = \"rules_x_extra\", version = \"1.0\")\n",
				"deps.bzl":     "def deps():\n    http_archive(\n        name = \"zlib\",\n        sha256 = \"old\",\n        urls = [\"https://a/zlib-1.2.tar.gz\", \"https://b/zlib-1.2.tar.gz\"],\n    )\n",
			},
			script: `load("@fleet//bazel.star", "set_archive", "set_dep", "set_version", "version")
def change(repo):
    set_dep(repo, "rules_x", "2.0")
    set_archive(repo, "zlib", ["https://a/zlib-1.3.tar.gz"], "abc123")
    repo.set_var("v", set_version(repo, "7.4.1") or version(repo) != None)
`,
			want: map[string]string{
				"MODULE.bazel": "bazel_dep(version = \"2.0\", name = \"rules_x\", dev_dependency = True)\nbazel_dep(name = \"rules_x_extra\", version = \"1.0\")\n",
				"deps.bzl":     "def deps():\n    http_archive(\n        name = \"zlib\",\n        sha256 = \"abc123\",\n        urls = [\"https://a/zlib-1.3.tar.gz\"],\n    )\n",
			},
			vars: map[string]string{"v": "false"},
		},
		"sbt and php": {
			repo: map[string]string{
				"build.sbt":           "libraryDependencies += \"io.circe\" %%% \"circe-core\" % \"0.14.5\"\nscalaVersion := \"3.3.1\"\n",
				"project/plugins.sbt": "addSbtPlugin(\"org.scalameta\" % \"sbt-scalafmt\" % \"2.5.0\")\n",
				"composer.json":       "{\"require-dev\": {\"phpunit/phpunit\": \"^10\"}}",
			},
			script: `load("@fleet//sbt.star", "set_dependency", "set_sbt_version", "set_scala_version")
load("@fleet//php.star", "set_php", "set_require")
def change(repo):
    set_dependency(repo, "io.circe", "circe-core", "0.14.10")
    set_dependency(repo, "org.scalameta", "sbt-scalafmt", "2.5.2")
    set_scala_version(repo, "3.3.4")
    repo.set_var("props", set_sbt_version(repo, "1.10.2"))
    set_require(repo, "phpunit/phpunit", "^11")
    repo.set_var("absent", set_require(repo, "monolog/monolog", "^3"))
    repo.set_var("php", set_php(repo, "^8.3"))
`,
			want: map[string]string{
				"build.sbt":           "libraryDependencies += \"io.circe\" %%% \"circe-core\" % \"0.14.10\"\nscalaVersion := \"3.3.4\"\n",
				"project/plugins.sbt": "addSbtPlugin(\"org.scalameta\" % \"sbt-scalafmt\" % \"2.5.2\")\n",
				"composer.json":       "{\"require-dev\": {\"phpunit/phpunit\": \"^11\"}}\n",
			},
			vars: map[string]string{"props": "false", "absent": "[]", "php": "[]"},
		},
		"apple and elixir": {
			repo: map[string]string{
				"Package.swift": "// swift-tools-version: 5.10\n.package(name: \"Log\", url: \"https://github.com/apple/swift-log\", exact: \"1.5.0\"),\n.package(url: \"https://github.com/apple/swift-nio.git\", branch: \"main\"),\n",
				"Podfile":       "platform :ios, \"16.0\"\npod \"Alamofire\", \"~> 5.6\"\npod 'Kit', :git => 'https://x/kit.git'\n",
				"mix.exs":       "[{:plug, \"~> 1.14\"}, {:lib, git: \"https://x/lib.git\"}]\n",
			},
			script: `load("@fleet//swift.star", "set_package", "set_tools_version")
load("@fleet//cocoapods.star", "set_platform", "set_pod")
load("@fleet//elixir.star", "set_dep")
def change(repo):
    set_package(repo, "https://github.com/apple/swift-log.git", "1.6.1")
    repo.set_var("nio", set_package(repo, "https://github.com/apple/swift-nio", "2.0.0"))
    repo.set_var("tools", set_tools_version(repo, "5.9"))
    set_pod(repo, "Alamofire", "~> 5.9")
    repo.set_var("kit", set_pod(repo, "Kit", "1.0"))
    repo.set_var("platform", set_platform(repo, "ios", "15.0"))
    set_dep(repo, "plug", "~> 1.16")
    repo.set_var("lib", set_dep(repo, "lib", "1.0"))
`,
			want: map[string]string{
				"Package.swift": "// swift-tools-version: 5.10\n.package(name: \"Log\", url: \"https://github.com/apple/swift-log\", exact: \"1.6.1\"),\n.package(url: \"https://github.com/apple/swift-nio.git\", branch: \"main\"),\n",
				"Podfile":       "platform :ios, \"16.0\"\npod \"Alamofire\", \"~> 5.9\"\npod 'Kit', :git => 'https://x/kit.git'\n",
				"mix.exs":       "[{:plug, \"~> 1.16\"}, {:lib, git: \"https://x/lib.git\"}]\n",
			},
			vars: map[string]string{"nio": "[]", "tools": "[]", "kit": "[]", "platform": "[]", "lib": "[]"},
		},
		"buf, npm, go.work": {
			repo: map[string]string{
				"proto/buf.gen.yaml": "version: v1\nplugins:\n  - plugin: buf.build/grpc/go:v1.3.0\n    out: gen\n",
				"proto/buf.yaml":     "version: v2\ndeps:\n  - \"buf.build/googleapis/googleapis\"  # google apis\n",
				"package.json":       "{\"name\": \"x\"}",
			},
			script: `load("@fleet//buf.star", "set_dep_ref", "set_plugin_version")
load("@fleet//npm.star", "set_npmrc", "set_package_manager")
load("@fleet//gomod.star", "set_work_go_version")
def change(repo):
    set_plugin_version(repo, "buf.build/grpc/go", "v1.5.1")
    set_dep_ref(repo, "buf.build/googleapis/googleapis", "abc")
    repo.set_var("pm", set_package_manager(repo, "pnpm@9.12.0"))
    set_npmrc(repo, "save-exact", True)
    repo.set_var("work", set_work_go_version(repo, "1.27"))
`,
			want: map[string]string{
				"proto/buf.gen.yaml": "version: v1\nplugins:\n  - plugin: buf.build/grpc/go:v1.5.1\n    out: gen\n",
				"proto/buf.yaml":     "version: v2\ndeps:\n  - \"buf.build/googleapis/googleapis:abc\"  # google apis\n",
				".npmrc":             "save-exact=true\n",
			},
			vars: map[string]string{"pm": "false", "work": "false"},
		},
	} {
		runLib(t, name, tc)
	}
	_, _, _, err := run(t, "load('@fleet//bazel.star', 'set_archive')\ndef change(repo):\n  set_archive(repo, 'x', 'u', 'not-hex')\n", nil, RepoInfo{})
	if err == nil || !strings.Contains(err.Error(), "is not a hex digest") {
		t.Errorf("bad sha256: %v", err)
	}
}
