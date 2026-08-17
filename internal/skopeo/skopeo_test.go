package skopeo

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestRepoPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"nginx:latest", "nginx:latest"},
		{"library/nginx:latest", "library/nginx:latest"},
		{"docker.io/library/nginx:latest", "library/nginx:latest"},
		{"gcr.io/my-project/app:1.0", "my-project/app:1.0"},
		{"localhost:5000/foo/bar", "foo/bar"},
	}
	for _, c := range cases {
		if got := RepoPath(c.in); got != c.want {
			t.Errorf("RepoPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestArchiveFileName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"nginx:latest", "nginx-latest.tar"},
		{"docker.io/library/hello-world:latest", "library_hello-world-latest.tar"},
		{"library/nginx", "library_nginx.tar"},
		{"my.registry.io/team/app:v1.2.3", "team_app-v1.2.3.tar"},
	}
	for _, c := range cases {
		if got := ArchiveFileName(c.in); got != c.want {
			t.Errorf("ArchiveFileName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildSkopeoArgs(t *testing.T) {
	cases := []struct {
		platform    string
		image       string
		archiveFile string
		want        []string
	}{
		{
			"", "nginx:latest", "/out/nginx-latest.tar",
			[]string{"copy", "--all", "docker://nginx:latest", "oci-archive:/out/nginx-latest.tar"},
		},
		{
			"linux/amd64", "nginx:latest", "/out/x.tar",
			[]string{"copy", "--all", "--override-os", "linux", "--override-arch", "amd64", "docker://nginx:latest", "oci-archive:/out/x.tar"},
		},
		{
			"linux", "nginx:latest", "/out/x.tar",
			[]string{"copy", "--all", "--override-os", "linux", "docker://nginx:latest", "oci-archive:/out/x.tar"},
		},
	}
	for _, c := range cases {
		got := BuildSkopeoArgs(c.platform, c.image, c.archiveFile)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("BuildSkopeoArgs(%q, %q, %q) = %v, want %v", c.platform, c.image, c.archiveFile, got, c.want)
		}
	}
}

func TestIsVersionSupported(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"skopeo version 1.23.0", true},
		{"skopeo version 1.14.0", true},
		{"skopeo version 1.6.0", true},
		{"skopeo version 1.6.0 (commit: abc123, ...)", true},
		{"skopeo version 2.0.0", true},
		{"skopeo version 1.5.4", false},
		{"skopeo version 0.1.40", false},
		{"skopeo version 1.5", false},
		{"unexpected output", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsVersionSupported(c.line); got != c.want {
			t.Errorf("IsVersionSupported(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func TestCopyWhitelistedBinaries(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTestFile(t, filepath.Join(src, "skopeo"), "unix-bin", 0755)
	writeTestFile(t, filepath.Join(src, "skopeo.exe"), "win-bin", 0644)

	got, err := CopyWhitelistedBinaries(src, dst, []string{"skopeo", "skopeo.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"skopeo", "skopeo.exe"}) {
		t.Errorf("got %v, want [skopeo skopeo.exe]", got)
	}
	for _, name := range []string{"skopeo", "skopeo.exe"} {
		if _, err := os.Stat(filepath.Join(dst, name)); err != nil {
			t.Errorf("%s not copied: %v", name, err)
		}
	}
}

func TestCopyWhitelistedBinariesSkipsMissing(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTestFile(t, filepath.Join(src, "skopeo"), "unix-bin", 0755)

	got, err := CopyWhitelistedBinaries(src, dst, []string{"skopeo", "skopeo.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"skopeo"}) {
		t.Errorf("got %v, want [skopeo]", got)
	}
	if _, err := os.Stat(filepath.Join(dst, "skopeo.exe")); !os.IsNotExist(err) {
		t.Error("skopeo.exe should not have been copied")
	}
}

func TestCopyWhitelistedBinariesNoneMatch(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	got, err := CopyWhitelistedBinaries(src, dst, []string{"skopeo", "skopeo.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestCopyWhitelistedBinariesPreservesMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit is not meaningful on Windows")
	}
	src := t.TempDir()
	dst := t.TempDir()
	writeTestFile(t, filepath.Join(src, "skopeo"), "unix-bin", 0755)

	if _, err := CopyWhitelistedBinaries(src, dst, []string{"skopeo"}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dst, "skopeo"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0100 == 0 {
		t.Errorf("copied skopeo lost executable bit: %v", fi.Mode().Perm())
	}
}

func TestCopyWhitelistedBinariesSkipsExistingDest(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeTestFile(t, filepath.Join(src, "skopeo"), "new", 0755)
	writeTestFile(t, filepath.Join(dst, "skopeo"), "existing", 0644)

	if _, err := CopyWhitelistedBinaries(src, dst, []string{"skopeo"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "skopeo"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Errorf("existing dest was overwritten: %q", string(data))
	}
}
