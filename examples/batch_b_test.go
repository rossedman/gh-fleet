package examples_test

import (
	"strings"
	"testing"
)

func TestBatchBExamples(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{".bazelversion": "6.5.0\n", "MODULE.bazel": "bazel_dep(name = \"rules_go\", version = \"0.46.0\")\nbazel_dep(name = \"gazelle\", version = \"0.35.0\")\n"})
	runTwice(t, "bazel-module-bump", dir)
	contains(t, dir, ".bazelversion", "7.4.1")
	contains(t, dir, "MODULE.bazel", `version = "0.50.1"`, `version = "0.39.1"`)

	dir = scratch(t, map[string]string{"build.sbt": "scalaVersion := \"2.13.12\"\ncrossScalaVersions := Seq(\"2.13.12\", \"3.3.4\")\n", "project/build.properties": "sbt.version=1.9.9\n"})
	runTwice(t, "sbt-scala-bump", dir)
	contains(t, dir, "build.sbt", `scalaVersion := "2.13.15"`, `Seq("2.13.15", "3.3.4")`)
	contains(t, dir, "project/build.properties", "sbt.version=1.10.2")

	path, log := stub(t, "composer")
	dir = scratch(t, map[string]string{"composer.json": "{\n    \"require\": {\n        \"php\": \"^8.1\",\n        \"laravel/framework\": \"^10.10\"\n    }\n}\n", "composer.lock": "{}\n"})
	runScript(t, "php-laravel-11", dir, path)
	contains(t, dir, "composer.json", `"php": "^8.2"`, `"laravel/framework": "^11.0"`)
	if !strings.Contains(read(t, log), " update laravel/framework --with-all-dependencies --no-install --no-scripts") {
		t.Errorf("composer calls = %q", read(t, log))
	}

	dir = scratch(t, map[string]string{"Package.swift": "// swift-tools-version:5.8\nlet p = Package(dependencies: [.package(url: \"https://github.com/apple/swift-log.git\", from: \"1.5.3\")])\n"})
	runTwice(t, "swift-package-bump", dir)
	contains(t, dir, "Package.swift", "swift-tools-version:5.10", `from: "1.6.1"`)

	dir = scratch(t, map[string]string{"ios/Podfile": "platform :ios, '13.0'\ntarget 'App' do\n  pod 'Alamofire', '~> 5.6'\nend\n"})
	runTwice(t, "cocoapods-ios-15", dir)
	contains(t, dir, "ios/Podfile", "platform :ios, '15.0'", "pod 'Alamofire', '~> 5.9'")

	dir = scratch(t, map[string]string{"mix.exs": "[elixir: \"~> 1.14\", deps: [{:phoenix, \"~> 1.7.0\"}]]\n", ".tool-versions": "erlang 26.2\nelixir 1.15.7-otp-26\n"})
	runTwice(t, "elixir-phoenix", dir)
	contains(t, dir, "mix.exs", `elixir: "~> 1.17"`, `{:phoenix, "~> 1.7.14"}`)
	contains(t, dir, ".tool-versions", "erlang 27.1\nelixir 1.17.3-otp-27\n")

	dir = scratch(t, map[string]string{"buf.gen.yaml": "version: v2\nplugins:\n  - remote: buf.build/protocolbuffers/go:v1.31.0\n  - remote: buf.build/grpc/go:v1.3.0\n"})
	runTwice(t, "buf-plugins", dir)
	contains(t, dir, "buf.gen.yaml", "protocolbuffers/go:v1.35.1", "grpc/go:v1.5.1")

	dir = scratch(t, map[string]string{"package.json": "{\n  \"packageManager\": \"pnpm@8.15.4\"\n}\n", "pnpm-lock.yaml": "x\n"})
	runTwice(t, "node-pnpm-9", dir)
	contains(t, dir, "package.json", `"packageManager": "pnpm@9.12.0"`)
	contains(t, dir, ".npmrc", "engine-strict=true\n")
}
