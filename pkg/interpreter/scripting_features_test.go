package interpreter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

// ===========================================================================
// 1. Mutable Closures
// ===========================================================================

func TestMutableClosureCounter(t *testing.T) {
	result := evalInput(t, `
counter = 0
increment = fn() { counter = counter + 1 }
increment()
increment()
increment()
counter`)
	assertInteger(t, result, 3)
}

func TestMutableClosureSharedState(t *testing.T) {
	result := evalInput(t, `
x = 10
add = fn(n) { x = x + n }
sub = fn(n) { x = x - n }
add(5)
sub(3)
x`)
	assertInteger(t, result, 12)
}

func TestMutableClosureNestedFunctions(t *testing.T) {
	result := evalInput(t, `
def make_counter()
  count = 0
  def inc()
    count = count + 1
    count
  end
  inc
end
counter = make_counter()
counter()
counter()
counter()`)
	assertInteger(t, result, 3)
}

func TestLocalVariableDoesNotLeak(t *testing.T) {
	result := evalInput(t, `
x = "outer"
def test()
  y = "local"
  x
end
test()`)
	assertString(t, result, "outer")
}

func TestNewVariableInFunctionIsLocal(t *testing.T) {
	result := evalInput(t, `
def test()
  new_var = 42
  new_var
end
test()`)
	assertInteger(t, result, 42)
}

func TestMutableClosureInForLoop(t *testing.T) {
	result := evalInput(t, `
total = 0
for i in [1, 2, 3, 4, 5]
  total = total + i
end
total`)
	assertInteger(t, result, 15)
}

// ===========================================================================
// 2. String Formatting
// ===========================================================================

func TestFormatString(t *testing.T) {
	result := evalInput(t, `format("Hello, %s!", "Alice")`)
	assertString(t, result, "Hello, Alice!")
}

func TestFormatInteger(t *testing.T) {
	result := evalInput(t, `format("Count: %d", 42)`)
	assertString(t, result, "Count: 42")
}

func TestFormatFloat(t *testing.T) {
	result := evalInput(t, `format("Pi: %.2f", 3.14159)`)
	assertString(t, result, "Pi: 3.14")
}

func TestFormatPadded(t *testing.T) {
	result := evalInput(t, `format("%05d", 42)`)
	assertString(t, result, "00042")
}

func TestFormatHex(t *testing.T) {
	result := evalInput(t, `format("%x", 255)`)
	assertString(t, result, "ff")
}

func TestFormatMultipleArgs(t *testing.T) {
	result := evalInput(t, `format("%s is %d years old", "Alice", 30)`)
	assertString(t, result, "Alice is 30 years old")
}

// ===========================================================================
// 3. Typed Errors
// ===========================================================================

func TestTypedErrorCreation(t *testing.T) {
	result := evalInput(t, `
e = Error("not found")
e.message`)
	assertString(t, result, "not found")
}

func TestTypedErrorWithData(t *testing.T) {
	result := evalInput(t, `
e = Error("not found", {code: 404, path: "/tmp"})
e.data.code`)
	assertInteger(t, result, 404)
}

func TestTypedErrorType(t *testing.T) {
	result := evalInput(t, `
e = Error("oops")
e.type`)
	assertString(t, result, "Error")
}

func TestTypedErrorThrowCatch(t *testing.T) {
	result := evalInput(t, `
try
  throw Error("file not found", {code: 404})
catch e
  e.message
end`)
	assertString(t, result, "file not found")
}

func TestTypedErrorThrowCatchData(t *testing.T) {
	result := evalInput(t, `
try
  throw Error("fail", {code: 500})
catch e
  e.data.code
end`)
	assertInteger(t, result, 500)
}

func TestStringThrowStillWorks(t *testing.T) {
	result := evalInput(t, `
try
  throw "plain error"
catch e
  e
end`)
	assertString(t, result, "plain error")
}

// ===========================================================================
// 4. Enhanced Regex
// ===========================================================================

func TestRegexMatchGroups(t *testing.T) {
	result := evalInput(t, `
import "regex"
m = match_groups("(\\d{4})-(\\d{2})-(\\d{2})", "2026-02-06")
m[1]`)
	assertString(t, result, "2026")
}

func TestRegexMatchGroupsFull(t *testing.T) {
	result := evalInput(t, `
import "regex"
m = match_groups("(\\d{4})-(\\d{2})-(\\d{2})", "2026-02-06")
m[0]`)
	assertString(t, result, "2026-02-06")
}

func TestRegexMatchGroupsNoMatch(t *testing.T) {
	result := evalInput(t, `
import "regex"
match_groups("(\\d{4})-(\\d{2})", "hello")`)
	if _, ok := result.(*Nil); !ok {
		t.Errorf("expected Nil, got %T (%+v)", result, result)
	}
}

func TestRegexTest(t *testing.T) {
	result := evalInput(t, `
import "regex"
test("^\\d+$", "12345")`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolObj.Value {
		t.Error("expected true")
	}
}

func TestRegexTestFalse(t *testing.T) {
	result := evalInput(t, `
import "regex"
test("^\\d+$", "hello")`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if boolObj.Value {
		t.Error("expected false")
	}
}

func TestRegexSplitRegex(t *testing.T) {
	result := evalInput(t, `
import "regex"
parts = split_regex("\\s+", "hello   world   foo")
len(parts)`)
	assertInteger(t, result, 3)
}

func TestRegexSplitRegexValues(t *testing.T) {
	result := evalInput(t, `
import "regex"
parts = split_regex("\\s+", "hello   world   foo")
parts[2]`)
	assertString(t, result, "foo")
}

// ===========================================================================
// 5. OS Module -- Environment Variables
// ===========================================================================

func TestEnvGet(t *testing.T) {
	os.Setenv("VIBE_TEST_VAR", "hello_vibe")
	defer os.Unsetenv("VIBE_TEST_VAR")

	result := evalInput(t, `
import "os"
env("VIBE_TEST_VAR")`)
	assertString(t, result, "hello_vibe")
}

func TestEnvGetMissing(t *testing.T) {
	result := evalInput(t, `
import "os"
env("VIBE_NONEXISTENT_VAR_12345")`)
	if _, ok := result.(*Nil); !ok {
		t.Errorf("expected Nil, got %T", result)
	}
}

func TestSetEnv(t *testing.T) {
	defer os.Unsetenv("VIBE_SET_TEST")

	result := evalInput(t, `
import "os"
set_env("VIBE_SET_TEST", "works")
env("VIBE_SET_TEST")`)
	assertString(t, result, "works")
}

func TestEnvAll(t *testing.T) {
	os.Setenv("VIBE_ENV_ALL_TEST", "yes")
	defer os.Unsetenv("VIBE_ENV_ALL_TEST")

	result := evalInput(t, `
import "os"
all = env_all()
type(all)`)
	assertString(t, result, "HASH")
}

// ===========================================================================
// 6. OS Module -- File I/O
// ===========================================================================

func TestAppendFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "append_test.txt")
	if err := os.WriteFile(tmpFile, []byte("line1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := evalInput(t, `
import "os"
append_file("`+tmpFile+`", "line2\n")
read_file("`+tmpFile+`")`)
	assertString(t, result, "line1\nline2\n")
}

func TestDeleteFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "delete_test.txt")
	if err := os.WriteFile(tmpFile, []byte("temp"), 0644); err != nil {
		t.Fatal(err)
	}

	evalInput(t, `
import "os"
delete_file("`+tmpFile+`")`)

	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestListDir(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	result := evalInput(t, `
import "os"
files = list_dir("`+tmpDir+`")
len(files)`)
	assertInteger(t, result, 2)
}

func TestDirExists(t *testing.T) {
	result := evalInput(t, `
import "os"
dir_exists("/tmp")`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolObj.Value {
		t.Error("expected true")
	}
}

func TestDirExistsFalse(t *testing.T) {
	result := evalInput(t, `
import "os"
dir_exists("/nonexistent_dir_12345")`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if boolObj.Value {
		t.Error("expected false")
	}
}

func TestMkdirAndRemove(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "newdir")

	result := evalInput(t, `
import "os"
mkdir("`+tmpDir+`")
dir_exists("`+tmpDir+`")`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolObj.Value {
		t.Error("expected true")
	}
}

func TestMkdirP(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "a", "b", "c")

	result := evalInput(t, `
import "os"
mkdir_p("`+tmpDir+`")
dir_exists("`+tmpDir+`")`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolObj.Value {
		t.Error("expected true")
	}
}

func TestGetcwd(t *testing.T) {
	result := evalInput(t, `
import "os"
cwd = getcwd()
type(cwd)`)
	assertString(t, result, "STRING")
}

// ===========================================================================
// 7. OS Module -- Path Operations
// ===========================================================================

func TestPathBase(t *testing.T) {
	result := evalInput(t, `
import "os"
path_base("/home/user/file.txt")`)
	assertString(t, result, "file.txt")
}

func TestPathDir(t *testing.T) {
	result := evalInput(t, `
import "os"
path_dir("/home/user/file.txt")`)
	assertString(t, result, "/home/user")
}

func TestPathExt(t *testing.T) {
	result := evalInput(t, `
import "os"
path_ext("file.txt")`)
	assertString(t, result, ".txt")
}

func TestPathJoin(t *testing.T) {
	result := evalInput(t, `
import "os"
path_join("home", "user", "file.txt")`)
	assertString(t, result, "home/user/file.txt")
}

// ===========================================================================
// 8. OS Module -- Process Execution
// ===========================================================================

func TestShell(t *testing.T) {
	result := evalInput(t, `
import "os"
output = shell("echo hello")
trim(output)`)
	assertString(t, result, "hello")
}

func TestExec(t *testing.T) {
	result := evalInput(t, `
import "os"
r = exec("echo hello")
r.status`)
	assertInteger(t, result, 0)
}

func TestExecStdout(t *testing.T) {
	result := evalInput(t, `
import "os"
r = exec("echo hello")
trim(r.stdout)`)
	assertString(t, result, "hello")
}

func TestExecFailure(t *testing.T) {
	result := evalInput(t, `
import "os"
r = exec("exit 42")
r.status`)
	assertInteger(t, result, 42)
}

// ===========================================================================
// 9. OS Module -- CLI Args
// ===========================================================================

func TestARGVDefault(t *testing.T) {
	result := evalInput(t, `
import "os"
len(ARGV)`)
	assertInteger(t, result, 0)
}

func TestSetArgs(t *testing.T) {
	interp := New()
	interp.SetArgs("test.vb", []string{"--verbose", "output.txt"})

	l := lexer.New(`import "os"
ARGV[0]`)
	p := parser.New(l)
	program := p.ParseProgram()
	result := interp.Eval(program)
	assertString(t, result, "--verbose")
}

func TestScriptName(t *testing.T) {
	interp := New()
	interp.SetArgs("my_script.vb", []string{})

	l := lexer.New(`SCRIPT_NAME`)
	p := parser.New(l)
	program := p.ParseProgram()
	result := interp.Eval(program)
	assertString(t, result, "my_script.vb")
}

// ===========================================================================
// 10. Time Module
// ===========================================================================

func TestTimeNow(t *testing.T) {
	result := evalInput(t, `
import "time"
now = Time.now()
now.year`)
	intResult, ok := result.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}
	if intResult.Value < 2025 {
		t.Errorf("expected year >= 2025, got %d", intResult.Value)
	}
}

func TestTimeFields(t *testing.T) {
	result := evalInput(t, `
import "time"
now = Time.now()
now.month >= 1 && now.month <= 12`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolObj.Value {
		t.Error("expected true")
	}
}

func TestTimeUnix(t *testing.T) {
	result := evalInput(t, `
import "time"
now = Time.now()
now.unix > 0`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolObj.Value {
		t.Error("expected true")
	}
}

func TestTimeMeasure(t *testing.T) {
	result := evalInput(t, `
import "time"
elapsed = Time.measure(fn() { 1 + 1 })
elapsed >= 0`)
	boolObj, ok := result.(*Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}
	if !boolObj.Value {
		t.Error("expected true")
	}
}

func TestFormatTime(t *testing.T) {
	result := evalInput(t, `
import "time"
now = Time.now()
formatted = now.format_time("2006")
type(formatted)`)
	assertString(t, result, "STRING")
}
