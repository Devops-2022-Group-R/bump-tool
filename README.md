# Bump tool
Bump tool tries to create the next semver version of your software by using the Github API.

## Installation
```sh
export GOSUMDB=off # Necessary as the repo owner is capitalized
go install github.com/Devops-2022-Group-R/bump-tool@latest
```

## Usage
Create a [Github API token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)

### With url
```
bump-tool --token <github-token> --url "https://github.com/Devops-2022-Group-R/itu-minitwit/pull/43"
```

### With params
```
bump-tool --token <github-token> --owner "Devops-2022-Group-R" --repo "itu-minitwit" --pr 43
```

By default a loggin statements are included, to just log the new version add `--shouldLog=false`
