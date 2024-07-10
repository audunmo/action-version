# 📌#️⃣ Easily pin Github Action versions from version strings to commit hashes

Using version numbers to control your github actions is a bad security practice, as the version tag can be reassigned, leaving you open to supply chain attacks. Docs often use version numbers, and it's time-consuming to have to go get all those commits and pasting in the correct versions. `action-version` will automatically do this for you, for workflow files and markdown documentation.

# Installation

Ensure `go` is installed on your system. Then simply run `go install github.com/audunmo/action-version`

## Usage

When ran in a folder with .md or .yaml files, action-version will look through those files for strings matching the pattern `uses: actions/checkout@v1` and replace them with the commit hash of the commit tagged with v1.

> [!NOTE]
> Action Version will only look at files in the same folder as it is ran in. It will _not_ recursively look through subfolders.

```bash
cd path/to/your/repo/.github/workflows

action-version
```
