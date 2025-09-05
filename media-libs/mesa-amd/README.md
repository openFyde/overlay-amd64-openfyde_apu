### mesa-amd dev and uprev process

mesa-amd follows a traditional [cros_workon](https://chromium.googlesource.com/chromiumos/docs/+/HEAD/developer_guide.md#making-changes-to-packages-whose-source-code-is-checked-into-chromium-os-git-repositories) workflow where changes are made directly to src/third_party/mesa-amd. This is different from the [mesa](https://chromium.googlesource.com/chromiumos/overlays/chromiumos-overlay/+/HEAD/media-libs/mesa/mesa-9999.ebuild) project, which follows the CROS_WORKON_MANUAL_UPREV workflow.

Working in mesa-amd takes an "upstream first" philosophy, and downstream changes represent technical debt and should be avoided. Commit messages for downstream changes in mesa-amd are [prefixed in a similar fashion](https://chromium.googlesource.com/chromiumos/docs/+/HEAD/kernel_development.md#commit-messages-summary-lines-chromium_upstream_fromlist_backport) as changes in the kernel and elsewhere in Chrome OS.

Refer to the [upstream documentation](https://docs.mesa3d.org/releasing.html#release-schedule) for details about upstream mesa's release schedule. To summarize, approximately every three months is a [feature release](https://docs.mesa3d.org/releasing.html#feature-releases), followed by [stable releases](https://docs.mesa3d.org/releasing.html#stable-releases) approximately every two weeks.

mesa-amd generally tracks upstream feature release branches, with possibly a small number of downstream changes applied on top.

Tags and branches from upstream are automatically synced to the chromiumos mesa mirror. Tags and branch names are synced under upstream/... (for example, [upstream/mesa-21.2.1](https://chromium.googlesource.com/chromiumos/third_party/mesa/+/refs/tags/upstream/mesa-21.2.1) and [upstream/21.1](https://chromium.googlesource.com/chromiumos/third_party/mesa/+/refs/heads/upstream/21.2)).

#### Uprev process

When uprevving to a new release branch, follow the below process to reset the tree to its upstream state.
```
 # Create a merge commit that resolves all merge conflicts such that all changes are discarded in favor of those in the new upstream branch
 git merge upstream/mesa-21.2.1 -X theirs --no-commit
 git read-tree upstream/mesa-21.2.1
 # Restore this file to ensure that presubmit checks survive the uprev.
 git add PRESUBMIT.cfg
 # Create an appropriate commit message, and add BUG= and TEST= tags appropriately.
 git commit -m "CHROMIUM: Reset tree to upstream/mesa-21.2.1"
 # Clean up the index
 git clean -f
 git reset --hard HEAD
 # Confirm that the state of the tree is identical to the upstream state, modulo the PRESUMBIT.cfg that was added.
 git diff-tree --no-commit-id --name-status upstream/mesa-21.2.1 HEAD
```

Use `git log --first-parent --pretty='%h %s'` to see the list of changes since the last time the tree was reset, and assess whether downstream patches can be dropped, or if they need to be cherry-picked again. You may need to cherry-pick some long-lived dowstream changes (i.e. CHROMIUM: patches). When you cherry-pick such a patch, edit the commit message to strip old tags such as Change-Id, Reviewed-By, etc., and update the BUG= to reference the bug tracking the uprev effort.

Subsequent stable releases from upstream can be applied by using `git merge` and uploading the merge commit to Gerrit.


#### Downstream changes
The preferred flow to get changes into mesa-amd is to:
1. Send a change upstream for review
2. Get it merged upstream
3. Ensure it is included in the next upstream stable release
4. `git merge` the next stable release containing the change.

There may be cases where we cannot do this for a particular change, or a change is needed more urgently than this process allows. In such cases, we may upload changes to Gerrit with an appropriate prefix. These prefixes are:
- UPSTREAM: Indicates that the patch has landed in upstream mesa
  - This is appropriate when the UPSTREAM change will never land in a subsequent stable release, or we need to land it in Chrome OS more expediently than the upstream stable release process allows for.
  - Use `git cherry-pick -x` to ensure that the commit message contains the commit hash of the UPSTREAM commit
- BACKPORT: Same as upstream, but there are conflicts that needed to be addressed
  - Describe the conflicts and their resolution in the commit message
- FROMLIST: The patch is under review in an upstream merge request
  - These should not be merged except for P0/P1 issues. It's strongly preferred to wait for the completion of the review upstream, and then apply the change as UPSTREAM
  - Owner+Reviewer of this change is responsible to follow up on the upstream review process and ensure that it gets merged in a timely manner.
  - Include a link to the merge-request in the commit message
- CHROMIUM: The patch cannot be upstreamed
  - There must be strong justification for why an upstreamable solution is not viable.
