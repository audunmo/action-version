# Automatically change Github Action versions from semver strings to commit hashes

Having semver versions control your github actions is bad, and we shouldn't do it. But, docs always use semver versions, and it's annoying to have to go get all those commits and pasting in the correct versions. So I made this.

```
go install github.com/audunmo/action-version@latest # the irony here is not lost on me, but I have not yet made a proper release

cd your-repo/.github/workflows

action-version
```
