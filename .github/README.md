# Continuous integration

CI runs on pushes, pull requests, manual dispatch and weekly scheduled scans. Actions
are pinned to commit hashes. CodeQL uses the security-extended query suite. Test
failures and coverage regressions fail CI; reports are retained as workflow artifacts.

Dependabot checks gomod dependencies and GitHub Actions weekly. Merge its configuration
into the default branch to activate it. Dependency PRs run the same checks and require
review before merging.

`branch-ruleset.json` is a proposed repository ruleset requiring PRs, resolved review
threads, an up-to-date branch and both CI jobs. Importing/applying it in repository
settings is separate from merging these files; this change does not enable it remotely.
