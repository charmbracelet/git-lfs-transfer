# Git LFS Transfer

![](https://stuff.charm.sh/soft-serve/git-lfs-transfer.png)

A server-side implementation of the [Git LFS pure SSH-based protocol proposal](https://github.com/git-lfs/git-lfs/blob/main/docs/proposals/ssh_adapter.md).

`git-lfs-transfer` transfers large files stored in Git over SSH.

## Installation

```bash
go install github.com/charmbracelet/git-lfs-transfer@latest
```

## Usage

```bash
# Usage
git-lfs-transfer <Directory> <Operation>

# Example
git-lfs-transfer repo.git upload
git-lfs-transfer repo.git download
```

## Acknowledgements

This library implements the [Git LFS pure SSH-based protocol proposal](https://github.com/git-lfs/git-lfs/blob/main/docs/proposals/ssh_adapter.md).

This library is ported from [Brian Carlson](https://github.com/bk2204)'s
library, [`scrutiger`](https://github.com/bk2204/scutiger), and has been
rewritten in Go.

## Feedback

We'd love to hear your thoughts on this project. Feel free to drop us a note!

* [Twitter](https://twitter.com/charmcli)
* [The Fediverse](https://mastodon.social/@charmcli)
* [Discord](https://charm.sh/chat)

## License

[MIT](https://github.com/charmbracelet/git-lfs-transfer/raw/master/LICENSE)

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source • نحنُ نحب المصادر المفتوحة
