package test_common

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func newTempFile() *os.File {
	f, err := os.CreateTemp("", "test")
	if err != nil {
		panic(err)
	}
	return f
}

func closeAndDeleteTempFile(f *os.File) {
	err := f.Close()
	if err != nil {
	}
	err = os.Remove(f.Name())
	if err != nil {
		panic(err)
	}
}

type TestResult struct {
	StdOut              string
	StdErr              string
	NonEmptyStdOutLines []string
	NonEmptyStdErrLines []string
}

func (t *TestResult) HasMatchingLine(pattern string) bool {
	for _, l := range t.NonEmptyStdOutLines {
		if strings.Contains(l, pattern) {
			return true
		}
	}
	for _, l := range t.NonEmptyStdErrLines {
		if strings.Contains(l, pattern) {
			return true
		}
	}
	return false
}

type TestSpec struct {
	Name     string
	Args     []string
	Expected string
}

var testMutex = sync.Mutex{}

func RunSingleTest(mainFn func(), args ...string) TestResult {

	// Only one test at a time, since we are replacing stdout and stderr globally
	testMutex.Lock()
	defer testMutex.Unlock()

	preArgs := os.Args
	preStdOut := os.Stdout
	preStdErr := os.Stderr

	stdOutCaptured := newTempFile()
	defer closeAndDeleteTempFile(stdOutCaptured)

	stdErrCaptured := newTempFile()
	defer closeAndDeleteTempFile(stdErrCaptured)

	os.Stdout = stdOutCaptured
	os.Stderr = stdErrCaptured

	os.Args = append([]string{"test-app"}, args...)
	defer func() {
		os.Args = preArgs
		os.Stdout = preStdOut
		os.Stderr = preStdErr
	}()

	// Run the actual test
	mainFn()

	// Go back to the start of the files
	_, _ = stdOutCaptured.Seek(0, 0)
	_, _ = stdErrCaptured.Seek(0, 0)

	// read back stdout
	stdOutBytes, err := io.ReadAll(stdOutCaptured)
	if err != nil {
		panic(err)
	}

	// read back stderr
	stdErrBytes, err := io.ReadAll(stdErrCaptured)
	if err != nil {
		panic(err)
	}

	stdOutStr := string(stdOutBytes)
	stdErrStr := string(stdErrBytes)

	result := TestResult{
		StdOut:              stdOutStr,
		StdErr:              stdErrStr,
		NonEmptyStdOutLines: NonEmptyLines(stdOutStr),
		NonEmptyStdErrLines: NonEmptyLines(stdErrStr),
	}

	return result
}

func NonEmptyLines(str string) []string {
	lines := []string{}
	for _, line := range strings.Split(str, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func RunTests(t *testing.T, mainFn func(), tests []TestSpec) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			res := RunSingleTest(mainFn, tt.Args...)
			if !res.HasMatchingLine(tt.Expected) {
				t.Errorf("Expected %s in output", tt.Expected)

				fmt.Printf("actual output:\n")
				for _, l := range res.NonEmptyStdErrLines {
					fmt.Printf("%s\n", l)
				}
				for _, l := range res.NonEmptyStdOutLines {
					fmt.Printf("%s\n", l)
				}
			}
		})
	}
}
