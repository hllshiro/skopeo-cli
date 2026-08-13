package skopeo

import (
	"reflect"
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
