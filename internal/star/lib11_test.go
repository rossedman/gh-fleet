package star

import "testing"

// TestLibNeverLowers runs version setters on repos that are already
// newer: every file must stay as it was.
func TestLibNeverLowers(t *testing.T) {
	t.Parallel()
	repo := map[string]string{
		".bazelversion":                   "9.2.0\n",
		"MODULE.bazel":                    "bazel_dep(name = \"gazelle\", version = \"0.51.3\")\n",
		"project/build.properties":        "sbt.version=2.0.4\n",
		"build.sbt":                       "libraryDependencies += \"org.typelevel\" %% \"cats-core\" % \"2.13.0\"\n",
		"Package.swift":                   ".package(url: \"https://github.com/apple/swift-log.git\", from: \"1.14.0\"),\n",
		"build.gradle.kts":                "plugins {\n  id(\"com.android.application\") version \"9.1.0\"\n  kotlin(\"android\") version \"2.3.0\"\n}\ndependencies {\n  implementation(\"com.squareup.okhttp3:okhttp:5.1.0\")\n}\n",
		"Cargo.toml":                      "[dependencies]\nserde = \"1.0.230\"\n",
		"CMakeLists.txt":                  "FetchContent_Declare(fmt GIT_REPOSITORY https://github.com/fmtlib/fmt GIT_TAG 12.0.0)\n",
		"Podfile":                         "pod 'Alamofire', '~> 6.0'\n",
		".devcontainer/devcontainer.json": "{\n  \"features\": {\"ghcr.io/devcontainers/features/node:2\": {}}\n}\n",
		"app.csproj":                      "<PackageReference Include=\"Serilog\" Version=\"5.0.0\" />\n",
		"global.json":                     "{\"sdk\": {\"version\": \"10.0.100\"}}\n",
		"mix.exs":                         "elixir: \"~> 1.18\",\n{:phoenix, \"~> 1.8\"}\n",
		"pom.xml":                         "<properties>\n<jackson.version>3.0.0</jackson.version>\n</properties>\n<dependency>\n<groupId>com.fasterxml.jackson.core</groupId>\n<artifactId>jackson-databind</artifactId>\n<version>${jackson.version}</version>\n</dependency>\n",
		"package.json":                    "{\n  \"packageManager\": \"pnpm@12.11.2\",\n  \"dependencies\": {\"lodash\": \"^5.0.0\"}\n}\n",
		"composer.json":                   "{\n  \"require\": {\"php\": \"^8.3\", \"laravel/framework\": \"^13.17\"}\n}\n",
		"requirements.txt":                "django==5.2.1\n",
		"pyproject.toml":                  "[project]\nrequires-python = \">=3.12\"\n",
		"Gemfile":                         "ruby \"3.4.1\"\ngem \"rails\", \"~> 8.0\"\n",
		"Gemfile.lock":                    "BUNDLED WITH\n   2.6.2\n",
		".rubocop.yml":                    "AllCops:\n  TargetRubyVersion: 3.4\n",
		"x.gemspec":                       "  s.required_ruby_version = \">= 3.2\"\n",
		".nvmrc":                          "lts/*\n",
		".tool-versions":                  "nodejs 24.1.0\n",
		"main.tf":                         "terraform {\n  required_version = \">= 1.9\"\n  required_providers {\n    aws = { source = \"hashicorp/aws\", version = \"~> 6.0\" }\n  }\n}\n",
		"build.pkr.hcl":                   "packer {\n  required_plugins {\n    amazon = { version = \">= 1.4.0\", source = \"github.com/hashicorp/amazon\" }\n  }\n}\n",
		"chart/Chart.yaml":                "version: 2.0.0\nappVersion: 3.0.0\ndependencies:\n  - name: redis\n    version: 21.0.0\n",
		"apps/app.yaml":                   "kind: Application\nspec:\n  source:\n    repoURL: https://github.com/acme/manifests\n    targetRevision: v3.0.0\n",
		".github/workflows/ci.yml":        "steps:\n  - uses: actions/checkout@v7\n",
		".pre-commit-config.yaml":         "repos:\n  - repo: https://github.com/psf/black\n    rev: 25.1.0\n",
		"buf.gen.yaml":                    "version: v2\nplugins:\n  - remote: buf.build/protocolbuffers/go:v1.36.5\n",
		"requirements.yml":                "collections:\n  - name: community.general\n    version: 10.0.0\n",
	}
	script := `load("@fleet//bazel.star", bazel_version = "set_version", "set_dep")
load("@fleet//sbt.star", "set_sbt_version", sbt_dep = "set_dependency")
load("@fleet//swift.star", "set_package")
load("@fleet//android.star", "set_agp_version")
load("@fleet//gradle.star", "set_kotlin_version", gradle_dep = "set_dependency")
load("@fleet//cargo.star", cargo_dep = "set_dependency")
load("@fleet//cmake.star", "set_fetchcontent_tag")
load("@fleet//cocoapods.star", "set_pod")
load("@fleet//devcontainer.star", "set_feature_version")
load("@fleet//dotnet.star", dotnet_pkg = "set_package", "set_sdk_version")
load("@fleet//elixir.star", elixir_dep = "set_dep", "set_elixir")
load("@fleet//maven.star", maven_dep = "set_dependency")
load("@fleet//npm.star", npm_dep = "set_dependency", "set_package_manager")
load("@fleet//php.star", "set_php", "set_require")
load("@fleet//python.star", "set_requirement", "set_requires_python")
load("@fleet//ruby.star", "set_bundler", "set_gem", "set_required_ruby", "set_rubocop_target", "set_ruby")
load("@fleet//runtime.star", runtime_version = "set_version")
load("@fleet//terraform.star", "set_provider_version", "set_required_version")
load("@fleet//packer.star", "set_plugin_version")
load("@fleet//helm.star", "set_app_version", "set_chart_version", helm_dep = "set_dependency")
load("@fleet//argocd.star", "set_target_revision")
load("@fleet//gha.star", "bump_action")
load("@fleet//precommit.star", "set_rev")
load("@fleet//buf.star", buf_plugin = "set_plugin_version")
load("@fleet//ansible.star", ansible_version = "set_version")
def change(repo):
    bazel_version(repo, "7.4.1")
    set_dep(repo, "gazelle", "0.39.1")
    set_sbt_version(repo, "1.10.2")
    sbt_dep(repo, "org.typelevel", "cats-core", "2.12.0")
    set_package(repo, "https://github.com/apple/swift-log.git", "1.6.1")
    set_agp_version(repo, "8.7.0")
    set_kotlin_version(repo, "2.0.21")
    gradle_dep(repo, "com.squareup.okhttp3:okhttp", "4.12.0")
    cargo_dep(repo, "serde", "1.0.210")
    set_fetchcontent_tag(repo, "fmt", "11.0.2")
    set_pod(repo, "Alamofire", "~> 5.9")
    set_feature_version(repo, "ghcr.io/devcontainers/features/node", "1")
    dotnet_pkg(repo, "Serilog", "4.0.0")
    set_sdk_version(repo, "8.0.400")
    elixir_dep(repo, "phoenix", "~> 1.7")
    set_elixir(repo, "~> 1.15")
    maven_dep(repo, "com.fasterxml.jackson.core", "jackson-databind", "2.18.0")
    npm_dep(repo, "lodash", "^4.17.21")
    set_package_manager(repo, "pnpm@9.12.0")
    set_require(repo, "laravel/framework", "^11.0")
    set_php(repo, "^8.2")
    set_requirement(repo, "django", "==4.2.16")
    set_requires_python(repo, ">=3.9")
    set_ruby(repo, "3.3.5")
    set_gem(repo, "rails", "~> 7.2")
    set_bundler(repo, "2.5.0")
    set_rubocop_target(repo, "3.3")
    set_required_ruby(repo, ">= 3.1")
    runtime_version(repo, "node", "22")
    set_required_version(repo, ">= 1.5")
    set_provider_version(repo, "hashicorp/aws", "~> 5.0")
    set_plugin_version(repo, "github.com/hashicorp/amazon", ">= 1.3.3")
    set_chart_version(repo, "chart", "1.2.0")
    set_app_version(repo, "chart", "2.0.0")
    helm_dep(repo, "redis", "20.2.1")
    set_target_revision(repo, "https://github.com/acme/manifests", "v2.1.0")
    bump_action(repo, "actions/checkout", "v4")
    set_rev(repo, "https://github.com/psf/black", "24.10.0")
    buf_plugin(repo, "protocolbuffers/go", "v1.35.1")
    ansible_version(repo, "community.general", "9.5.0")
`
	runLib(t, "never lowers", libCase{repo: repo, script: script, want: repo})
}
