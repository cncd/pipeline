package compiler

import (
	"testing"
)

func TestTrimImage(t *testing.T) {
	testdata := []struct {
		from string
		want string
	}{
		{
			from: "golang",
			want: "golang",
		},
		{
			from: "golang:latest",
			want: "golang",
		},
		{
			from: "golang:1.0.0",
			want: "golang",
		},
		{
			from: "library/golang",
			want: "golang",
		},
		{
			from: "library/golang:latest",
			want: "golang",
		},
		{
			from: "library/golang:1.0.0",
			want: "golang",
		},
		{
			from: "index.docker.io/library/golang:1.0.0",
			want: "golang",
		},
		{
			from: "gcr.io/library/golang:1.0.0",
			want: "gcr.io/library/golang",
		},
		// error cases, return input unmodified
		{
			from: "foo/bar?baz:boo",
			want: "foo/bar?baz:boo",
		},
	}
	for _, test := range testdata {
		got, want := trimImage(test.from), test.want
		if got != want {
			t.Errorf("Want image %q trimmed to %q, got %q", test.from, want, got)
		}
	}
}

func TestExpandImage(t *testing.T) {
	testdata := []struct {
		from string
		want string
	}{
		{
			from: "golang",
			want: "golang:latest",
		},
		{
			from: "golang:latest",
			want: "golang:latest",
		},
		{
			from: "golang:1.0.0",
			want: "golang:1.0.0",
		},
		{
			from: "library/golang",
			want: "golang:latest",
		},
		{
			from: "library/golang:latest",
			want: "golang:latest",
		},
		{
			from: "library/golang:1.0.0",
			want: "golang:1.0.0",
		},
		{
			from: "index.docker.io/library/golang:1.0.0",
			want: "golang:1.0.0",
		},
		{
			from: "gcr.io/golang",
			want: "gcr.io/golang:latest",
		},
		{
			from: "gcr.io/golang:1.0.0",
			want: "gcr.io/golang:1.0.0",
		},
		// error cases, return input unmodified
		{
			from: "foo/bar?baz:boo",
			want: "foo/bar?baz:boo",
		},
	}
	for _, test := range testdata {
		got, want := expandImage(test.from), test.want
		if got != want {
			t.Errorf("Want image %q expanded to %q, got %q", test.from, want, got)
		}
	}
}

func TestMatchImage(t *testing.T) {
	testdata := []struct {
		from, to string
		want     bool
	}{
		{
			from: "golang",
			to:   "golang",
			want: true,
		},
		{
			from: "golang:latest",
			to:   "golang",
			want: true,
		},
		{
			from: "library/golang:latest",
			to:   "golang",
			want: true,
		},
		{
			from: "index.docker.io/library/golang:1.0.0",
			to:   "golang",
			want: true,
		},
		{
			from: "golang",
			to:   "golang:latest",
			want: false,
		},
		{
			from: "library/golang:latest",
			to:   "library/golang",
			want: false,
		},
		{
			from: "gcr.io/golang",
			to:   "gcr.io/golang",
			want: true,
		},
		{
			from: "gcr.io/golang:1.0.0",
			to:   "gcr.io/golang",
			want: true,
		},
		{
			from: "gcr.io/golang:latest",
			to:   "gcr.io/golang",
			want: true,
		},
		{
			from: "gcr.io/golang",
			to:   "gcr.io/golang:latest",
			want: false,
		},
	}
	for _, test := range testdata {
		got, want := matchImage(test.from, test.to), test.want
		if got != want {
			t.Errorf("Want image %q matching %q is %v", test.from, test.to, want)
		}
	}
}
