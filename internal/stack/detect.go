package stack

import "os"

// ProjectStack holds the detected build/test/lint commands for the current project.
type ProjectStack struct {
	Language string
	BuildCmd string
	TestCmd  string
	LintCmd  string
}

// Detect inspects the current directory for known config files and returns the project stack.
func Detect() ProjectStack {
	switch {
	case exists("go.mod"):
		return ProjectStack{"go", "go build ./...", "go test ./...", "go vet ./..."}
	case exists("Cargo.toml"):
		return ProjectStack{"rust", "cargo check", "cargo test", "cargo clippy"}
	case exists("pom.xml"):
		return ProjectStack{"java", "mvn compile -q", "mvn test -q", "mvn checkstyle:check -q"}
	case exists("build.gradle"), exists("build.gradle.kts"):
		return ProjectStack{"java", "gradle compileJava -q", "gradle test -q", "gradle check -q"}
	case exists("package.json"):
		return detectNode()
	case exists("pyproject.toml"), exists("setup.py"):
		return ProjectStack{"python", "python -m py_compile", "pytest", "ruff check ."}
	case exists("Makefile"):
		return ProjectStack{"make", "make", "make test", "make lint"}
	default:
		return ProjectStack{}
	}
}

func detectNode() ProjectStack {
	s := ProjectStack{Language: "node", BuildCmd: "npx tsc --noEmit", TestCmd: "npm test", LintCmd: "npx eslint ."}
	if exists("tsconfig.json") {
		s.BuildCmd = "npx tsc --noEmit"
	}
	return s
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
