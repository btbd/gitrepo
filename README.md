# gitrepo

A simple tool for creating org repositories. It initializes the repo(s) with an Apache 2.0 license.

## Usage

`gitrepo -t TOKEN [OPTIONS] ORG/NAME,...`

Options:

- `t` - User API token (required)
- `a` - Comma separated users to add to the repo(s)
- `o` - Add users the organization as members in addition to adding them to the repo(s)
- `r` - Comma separated users to remove from the repo(s)
- `d` - Description of the repo(s).