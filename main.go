package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/briandowns/spinner"
	"github.com/google/go-github/v42/github"
	"golang.org/x/oauth2"
)

var shouldLog bool

func print(s string, args ...interface{}) {
	if shouldLog {
		log.Printf(s, args...)
	}
}

func createSpinner(text string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[0], 100*time.Millisecond)
	s.Suffix = text
	s.Start()

	return s
}

func getClient(token string) *github.Client {
	if shouldLog {
		s := createSpinner("Creating client")
		defer s.Stop()
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func getPullRequest(client *github.Client, owner string, repo string, prNumber int) (*github.PullRequest, error) {
	if shouldLog {
		s := createSpinner("Retrieving PR information")
		defer s.Stop()
	}

	pr, _, err := client.PullRequests.Get(context.Background(), owner, repo, prNumber)
	return pr, err
}

func getLatestReleaseTag(client *github.Client, owner string, repo string) (*github.RepositoryRelease, error) {
	if shouldLog {
		s := createSpinner("Retrieving latest release tag")
		defer s.Stop()
	}

	release, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		if v, ok := err.(*github.ErrorResponse); ok && v.Response.StatusCode == 404 {
			return nil, nil
		}
		return nil, err
	}

	return release, nil
}

func parseVersion(version string) *semver.Version {
	v, err := semver.NewVersion(version)
	if err != nil {
		ErrorHandle(err)
	}

	return v
}

func bumpVersion(version semver.Version, labels []string) semver.Version {
	major := false
	minor := false
	patch := false

	for _, label := range labels {
		switch strings.ToLower(label) {
		case "major":
			major = true
		case "minor":
			minor = true
		case "patch":
			patch = true
		}
	}

	if (major && minor) || (major && patch) || (minor && patch) {
		ErrorHandle(errors.New("cannot bump version with multiple labels"))
	}

	if major {
		return version.IncMajor()
	} else if minor {
		return version.IncMinor()
	} else if patch {
		return version.IncPatch()
	} else {
		ErrorHandle(errors.New("no version label found"))
		return version
	}
}

func main() {
	args := retrieveArgs()
	client := getClient(args.Token)

	pr, err := getPullRequest(client, args.RepoOwner, args.Repo, args.PrNumber)
	if err != nil {
		ErrorHandle(err)
	}

	labels := make([]string, len(pr.Labels))

	for _, label := range pr.Labels {
		labels = append(labels, *label.Name)
	}

	release, err := getLatestReleaseTag(client, args.RepoOwner, args.Repo)
	if err != nil {
		ErrorHandle(err)
	}

	var tag string
	if release == nil {
		print("No release found, assuming version 0.0.0\n")
		tag = "0.0.0"
	} else {
		tag = *release.TagName
	}

	print("Using latest release tag: %s\n", tag)
	version := *parseVersion(tag)

	newVersion := bumpVersion(version, labels)
	if shouldLog {
		log.Printf("New version: %s\n", newVersion.String())
	} else {
		fmt.Println(newVersion.String())
	}
}

type Args struct {
	Token     string
	PrNumber  int
	Repo      string
	RepoOwner string
}

func retrieveArgs() Args {
	args := Args{}

	token := flag.String("token", "", "Github token")

	prNumber := flag.Int("pr", -1, "Pull request url")
	repo := flag.String("repo", "", "Repo")
	owner := flag.String("owner", "", "Repo owner")
	url := flag.String("url", "", "Pull request url")
	lg := flag.Bool("shouldLog", true, "Should log output")

	flag.Parse()

	// Global var
	shouldLog = *lg

	if *token == "" {
		ErrorHandle(errors.New("no token provided"))
	}
	args.Token = *token

	if *prNumber == -1 && *repo == "" && *owner == "" {
		if *url == "" {
			ErrorHandle(errors.New("no pr url provided. provide either a pr number, repo and owner or a pr url"))
		} else {
			ar, err := retrievePrInfoFromUrl(*url, args)
			if err != nil {
				ErrorHandle(err)
			}

			args = ar
		}
	} else {
		args.PrNumber = *prNumber
		args.Repo = *repo
		args.RepoOwner = *owner
	}

	return args
}

func retrievePrInfoFromUrl(url string, args Args) (Args, error) {
	var re = regexp.MustCompile(`(?m).*\/(?P<owner>.+)\/(?P<repo>.+)\/pull\/(?P<prNumber>\d+)`)
	match := re.FindStringSubmatch(url)

	paramsMap := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}

	if paramsMap["repo"] == "" {
		return args, errors.New("no repo found")
	}
	args.Repo = paramsMap["repo"]

	if paramsMap["prNumber"] == "" {
		return args, errors.New("no pr number found")
	}
	if num, err := strconv.Atoi(paramsMap["prNumber"]); err != nil {
		return args, err
	} else {
		args.PrNumber = num
	}

	if paramsMap["owner"] == "" {
		return args, errors.New("no owner found")
	}
	args.RepoOwner = paramsMap["owner"]

	return args, nil
}

func ErrorHandle(err error) {
	log.Printf("Error occurred: %s\n", err)

	os.Exit(1)
}
