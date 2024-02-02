# Tree of Work

## How to build a release version

Run `./release.sh`. You will get a `tow` executable.
Add the executable's location to your PATH. Or move it to `/usr/local/bin` for example.

## How to run

`tow <directory of a bare repo>`

If you're already in a bare repo just run `tow .`

## How to debug

For development run in debug mode:

`DEBUG=1 go run . <path-to-bare-repo>`

And `tail debug.log` to see the logs.

## Output

```
Your worktrees: [1/22]



      Worktree       Branch         Modified at
> [ ] dummy-tree-18  dummy-tree-18  2024-01-18
  [ ] dummy-tree-19  dummy-tree-19  2024-01-18
  [ ] dummy-tree-20  dummy-tree-20  2024-01-18
  [ ] dummy-tree-21  dummy-tree-21  2024-01-18
  [ ] dummy-tree-22  dummy-tree-22  2024-01-18
  [ ] dummy-tree-23  dummy-tree-23  2024-01-18
  [ ] dummy-tree-24  dummy-tree-24  2024-01-18
  [ ] dummy-tree-25  dummy-tree-25  2024-01-18
  [ ] dummy-tree-26  dummy-tree-26  2024-01-18
  [ ] dummy-tree-27  dummy-tree-27  2024-01-18
  [ ] dummy-tree-28  dummy-tree-28  2024-01-18
  [ ] dummy-tree-29  dummy-tree-29  2024-01-18
  [ ] dummy-tree-3   dummy-tree-3   2024-01-18
  [ ] dummy-tree-30  dummy-tree-30  2024-01-18
  [ ] dummy-tree-31  dummy-tree-31  2024-01-18
  [ ] dummy-tree-32  dummy-tree-32  2024-01-18
  [ ] dummy-tree-33  dummy-tree-33  2024-01-18
  [ ] dummy-tree-34  dummy-tree-34  2024-01-18
  [ ] dummy-tree-35  dummy-tree-35  2024-01-18
  [ ] dummy-tree-36  dummy-tree-36  2024-01-18
  [ ] dummy-tree-37  dummy-tree-37  2024-01-18
  [ ] dummy-tree-38  dummy-tree-38  2024-01-18

q: Quit, Enter/Space: Select, d: Delete, D: Force Delete, r: Refresh
```
