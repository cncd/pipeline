package linter

import (
	"testing"

	"github.com/cncd/pipeline/pipeline/frontend/yaml"
	libcompose "github.com/docker/libcompose/yaml"
)

func TestLint(t *testing.T) {
	testdata := `
pipeline:
  build:
    image: docker
    privileged: true
    network_mode: host
    volumes:
      - /tmp:/tmp
    commands:
      - go build
      - go test
  publish:
    image: plugins/docker
    repo: foo/bar
services:
  redis:
    image: redis
    entrypoint: [ /bin/redis-server ]
    command: [ -v ]
`

	conf, err := yaml.ParseString(testdata)
	if err != nil {
		t.Fatalf("Cannot unmarshal yaml %q. Error: %s", testdata, err)
	}
	if err := New(WithTrusted(true)).Lint(conf); err != nil {
		t.Errorf("Expected lint returns no errors, got %q", err)
	}
}

func TestLintErrors(t *testing.T) {
	testdata := []struct {
		from string
		want string
	}{
		{
			from: "",
			want: "Invalid or missing pipeline section",
		},
		{
			from: "pipeline: { build: { image: '' }  }",
			want: "Invalid or missing image",
		},
		{
			from: "pipeline: { build: { image: golang, privileged: true }  }",
			want: "Insufficient privileges to use privileged mode",
		},
		{
			from: "pipeline: { build: { image: golang, shm_size: 10gb }  }",
			want: "Insufficient privileges to override shm_size",
		},
		{
			from: "pipeline: { build: { image: golang, dns: [ 8.8.8.8 ] }  }",
			want: "Insufficient privileges to use custom dns",
		},

		{
			from: "pipeline: { build: { image: golang, dns_search: [ example.com ] }  }",
			want: "Insufficient privileges to use dns_search",
		},
		{
			from: "pipeline: { build: { image: golang, devices: [ '/dev/tty0:/dev/tty0' ] }  }",
			want: "Insufficient privileges to use devices",
		},
		{
			from: "pipeline: { build: { image: golang, extra_hosts: [ 'somehost:162.242.195.82' ] }  }",
			want: "Insufficient privileges to use extra_hosts",
		},
		{
			from: "pipeline: { build: { image: golang, network_mode: host }  }",
			want: "Insufficient privileges to use network_mode",
		},
		{
			from: "pipeline: { build: { image: golang, networks: [ outside, default ] }  }",
			want: "Insufficient privileges to use networks",
		},
		{
			from: "pipeline: { build: { image: golang, volumes: [ '/opt/data:/var/lib/mysql' ] }  }",
			want: "Insufficient privileges to use volumes",
		},
		// cannot override entypoint, command for script steps
		{
			from: "pipeline: { build: { image: golang, commands: [ 'go build' ], entrypoint: [ '/bin/bash' ] } }",
			want: "Cannot override container entrypoint",
		},
		{
			from: "pipeline: { build: { image: golang, commands: [ 'go build' ], command: [ '/bin/bash' ] } }",
			want: "Cannot override container command",
		},
		// cannot override entypoint, command for plugin steps
		{
			from: "pipeline: { publish: { image: plugins/docker, repo: foo/bar, entrypoint: [ '/bin/bash' ] } }",
			want: "Cannot override container entrypoint",
		},
		{
			from: "pipeline: { publish: { image: plugins/docker, repo: foo/bar, command: [ '/bin/bash' ] } }",
			want: "Cannot override container command",
		},
	}

	for _, test := range testdata {
		conf, err := yaml.ParseString(test.from)
		if err != nil {
			t.Fatalf("Cannot unmarshal yaml %q. Error: %s", test.from, err)
		}

		lerr := New().Lint(conf)
		if lerr == nil {
			t.Errorf("Expected lint error for configuration %q", test.from)
		} else if lerr.Error() != test.want {
			t.Errorf("Want error %q, got %q", test.want, lerr.Error())
		}
	}
}

func TestLint_isScript(t *testing.T) {
	c := &yaml.Container{
		Commands: libcompose.Stringorslice{
			"go build",
			"go test",
		},
	}
	if isScript(c) != true {
		t.Errorf("Expect isScript returns true when container has commands")
	}
	if isScript(new(yaml.Container)) != false {
		t.Errorf("Expect isScript returns false when container has no commands")
	}
}

func TestLint_isPlugin(t *testing.T) {
	c := &yaml.Container{
		Vargs: map[string]interface{}{
			"foo": "bar",
			"baz": "qux",
		},
	}
	if isPlugin(c) != true {
		t.Errorf("Expect isPlugin returns true when container has vargs")
	}
	if isPlugin(new(yaml.Container)) != false {
		t.Errorf("Expect isPlugin returns false when container has no vargs")
	}
}

func TestLint_isService(t *testing.T) {
	c := &yaml.Container{
		Vargs: map[string]interface{}{
			"foo": "bar",
			"baz": "qux",
		},
	}
	if isService(c) != false {
		t.Errorf("Expect isService returns false when container has vargs")
	}
	c = &yaml.Container{
		Commands: libcompose.Stringorslice{
			"go build",
			"go test",
		},
	}
	if isService(c) != false {
		t.Errorf("Expect isService returns false when container has commands")
	}
	if isService(new(yaml.Container)) != true {
		t.Errorf("Expect isService returns true when container has no commands, no vargs")
	}
}
