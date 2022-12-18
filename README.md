# git-lfs-transfer

`git-lfs-transfer` is a server-side implementation of the proposed [Git LFS pure SSH-based protocol][proposal].
It is intended to be invoked over SSH to transfer Git LFS objects. This was
originally ported from [bk2204/scutiger](https://github.com/bk2204/scutiger) and
re-written in Go.

## Todo

- [x] Basic implementation of the proposed protocol
- [x] Locking support
- [ ] Integration tests

[proposal]: https://github.com/git-lfs/git-lfs/blob/main/docs/proposals/ssh_adapter.md

## License

[MIT](https://github.com/charmbracelet/git-lfs-transfer/raw/master/LICENSE)

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="The Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source • نحنُ نحب المصادر المفتوحة
