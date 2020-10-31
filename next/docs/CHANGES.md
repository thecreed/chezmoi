## Changes in v2, already done

General:
- `--recursive` is default for some commands, notably `chezmoi add`
- only diff format is git
- remove hg support
- remove source command (use git instead)
- `--include` option to many commands
- errors output to stderr, not stdout
- `--force` now global
- `--output` now global
- diff includes scripts
- archive includes scripts
- `encrypt` -> `encrypted` in chattr
- `--format` now global, don't use toml for dump
- `y`, `yes`, `on`, `n`, `no`, `off` recognized as bools
- added `promptBool`, `promptInt` functions to `chezmoi init`
- order for `merge` is now dest, target, source
- No more `--prompt` to `chezmoi edit`
- `--keep-going` global
- `chezmoi init` guesses your repo if you use github.com and dotfiles

Config file:
- rename `sourceVCS` to `git`
- use `gpg.recipient` instead of `gpgRecipient`
- rename `genericSecret` to `secret`
