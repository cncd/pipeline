package frontend

import "strconv"

type (
	// Metadata defines runtime m.
	Metadata struct {
		Repo Repo
		Curr Build
		Prev Build
		Job  Job
		Sys  System
	}

	// Repo defines runtime metadata for a repository.
	Repo struct {
		Name    string
		Link    string
		Remote  string
		Private bool
	}

	// Build defines runtime metadata for a build.
	Build struct {
		Number   int
		Created  int64
		Started  int64
		Finished int64
		Status   string
		Event    string
		Link     string
		Target   string

		Commit Commit
	}

	// Commit defines runtime metadata for a commit.
	Commit struct {
		Sha     string
		Ref     string
		Refspec string
		Branch  string
		Message string
		Author  Author
	}

	// Author defines runtime metadata for a commit author.
	Author struct {
		Name   string
		Email  string
		Avatar string
	}

	// Job defines runtime metadata for a job.
	Job struct {
		Number int
		Matrix map[string]string
	}

	// System defines runtime metadata for a ci/cd system.
	System struct {
		Name string
		Host string
		Link string
		Arch string
	}
)

// Environ returns the metadata as a map of environment variables.
func (m *Metadata) Environ() map[string]string {
	return map[string]string{
		"CI_REPO":                      m.Repo.Name,
		"CI_REPO_NAME":                 m.Repo.Name,
		"CI_REPO_LINK":                 m.Repo.Link,
		"CI_REPO_REMOTE":               m.Repo.Remote,
		"CI_REMOTE_URL":                m.Repo.Remote,
		"CI_REPO_PRIVATE":              strconv.FormatBool(m.Repo.Private),
		"CI_BUILD_NUMBER":              strconv.Itoa(m.Curr.Number),
		"CI_BUILD_CREATED":             strconv.FormatInt(m.Curr.Created, 10),
		"CI_BUILD_STARTED":             strconv.FormatInt(m.Curr.Started, 10),
		"CI_BUILD_FINISHED":            strconv.FormatInt(m.Curr.Finished, 10),
		"CI_BUILD_STATUS":              m.Curr.Status,
		"CI_BUILD_EVENT":               m.Curr.Event,
		"CI_BUILD_LINK":                m.Curr.Link,
		"CI_BUILD_TARGET":              m.Curr.Target,
		"CI_COMMIT_SHA":                m.Curr.Commit.Sha,
		"CI_COMMIT_REF":                m.Curr.Commit.Ref,
		"CI_COMMIT_REFSPEC":            m.Curr.Commit.Refspec,
		"CI_COMMIT_BRANCH":             m.Curr.Commit.Branch,
		"CI_COMMIT_MESSAGE":            m.Curr.Commit.Message,
		"CI_COMMIT_AUTHOR":             m.Curr.Commit.Author.Name,
		"CI_COMMIT_AUTHOR_NAME":        m.Curr.Commit.Author.Name,
		"CI_COMMIT_AUTHOR_EMAIL":       m.Curr.Commit.Author.Email,
		"CI_COMMIT_AUTHOR_AVATAR":      m.Curr.Commit.Author.Avatar,
		"CI_PREV_BUILD_NUMBER":         strconv.Itoa(m.Prev.Number),
		"CI_PREV_BUILD_CREATED":        strconv.FormatInt(m.Prev.Created, 10),
		"CI_PREV_BUILD_STARTED":        strconv.FormatInt(m.Prev.Started, 10),
		"CI_PREV_BUILD_FINISHED":       strconv.FormatInt(m.Prev.Finished, 10),
		"CI_PREV_BUILD_STATUS":         m.Prev.Status,
		"CI_PREV_BUILD_EVENT":          m.Prev.Event,
		"CI_PREV_BUILD_LINK":           m.Prev.Link,
		"CI_PREV_COMMIT_SHA":           m.Prev.Commit.Sha,
		"CI_PREV_COMMIT_REF":           m.Prev.Commit.Ref,
		"CI_PREV_COMMIT_REFSPEC":       m.Prev.Commit.Refspec,
		"CI_PREV_COMMIT_BRANCH":        m.Prev.Commit.Branch,
		"CI_PREV_COMMIT_MESSAGE":       m.Prev.Commit.Message,
		"CI_PREV_COMMIT_AUTHOR":        m.Prev.Commit.Author.Name,
		"CI_PREV_COMMIT_AUTHOR_NAME":   m.Prev.Commit.Author.Name,
		"CI_PREV_COMMIT_AUTHOR_EMAIL":  m.Prev.Commit.Author.Email,
		"CI_PREV_COMMIT_AUTHOR_AVATAR": m.Prev.Commit.Author.Avatar,
		"CI_JOB_NUMBER":                strconv.Itoa(m.Job.Number),
		"CI_SYSTEM":                    m.Sys.Name,
		"CI_SYSTEM_NAME":               m.Sys.Name,
		"CI_SYSTEM_LINK":               m.Sys.Link,
		"CI_SYSTEM_HOST":               m.Sys.Host,
		"CI_SYSTEM_ARCH":               m.Sys.Arch,
		"CI":                           m.Sys.Name,
	}
}
