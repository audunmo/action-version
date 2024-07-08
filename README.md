# 📌#️⃣ Easily pin Github Action versions from version strings to commit hashes

Using version numbers to control your github actions is a bad security practice, as the version tag can be reassigned, leaving you open to supply chain attacks. Docs often use version numbers, and it's time-consuming to have to go get all those commits and pasting in the correct versions. `action-version` will automatically do this for you, for workflow files and markdown documentation.

# Installation

Dependencies:

- Go (https://golang.org/doc/install)

With Go installed on your system run `go install github.com/audunmo/action-version@v1.0.0`

## Usage

When ran in a folder with .md or .yaml files, action-version will look through those files for strings matching the pattern `uses: actions/checkout@v4` and replace them with the commit hash of the commit tagged with v4.

```bash
cd path/to/your/repo/.github/workflows

# Update .yaml/.yml and .md files in the working directory
action-version

# Update .yaml/.yml and .md files in the working directory and in subfolders
action-version -r
```

## The problem

## Wait, why does it edit markdown files?

Github Actions `uses` strings are often copy-pasted from docs. Since docs often contain these version strings as opposed to commit hashes, the dangerous pattern of using version strings gets proliferated. Therefore, `action-version` will also edit any markdown files it sees. That way, consumers of your documentation can still get the benefits of a pinned version, with no extra effort for them

# Related projects

- (Mend Renovate)[https://github.com/apps/renovate]

Renovate is a really cool project that helps devs stay up-to-date with their dependecny version that helps devs stay up-to-date with their dependecny versions, including Github Actions versions. With some configuration, Renovate can perform similar tasks like `action-version`. You can see their docs here https://docs.renovatebot.com/modules/manager/github-actions/#additional-information

`action-version` is intended to fill the gap for where Renovate may be overkill for a project, or where you want to ensure that versions are pinned locally before they get pushed to your repo
