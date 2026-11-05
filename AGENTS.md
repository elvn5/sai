# Deployment rules

- Deploy production only from an existing GitHub Git tag (`vX.Y.Z`). Never deploy production directly from `main`, another branch, or an untagged commit. For a manual production workflow run, select the release tag and pass that same tag as the `tag` input.
- Deploy staging from `main` unless the user explicitly requests a different ref.
