# Nix image

## Build and test

From the repository root, run:

```sh
make image-nix
IMAGE=image-renderer-nix go test ./tests/acceptance/... -count=1
```

Docker with BuildKit runs Nix inside a Linux builder. On macOS, start Docker Desktop or OrbStack first. The build loads the image as `image-renderer-nix` and saves the archive to `dist/nix/<system>/image.tar.gz`. Use `NIX_IMAGE` to change the tag or `NIX_SYSTEM=x86_64-linux` or `aarch64-linux` to select the platform. Building for a different architecture requires Docker emulation. The Docker builder uses Docker's isolation with Nix's nested sandbox and syscall filtering disabled. On native Linux, use `nix build .#image` directly. For private CAs, see the [container configuration](../docs/sources/troubleshooting.md#nix-container).

[flake.lock](../flake.lock) pins Nixpkgs, including Go and Chromium. The image uses the upstream browser when the override is empty, equal, or older. A newer [chromium-info.json](chromium-info.json) selects the Nixpkgs Chromium recipe with the recorded source revisions and hashes. Nix uses a cached browser when available. Compiling it on a cache miss can take several hours.

The [Build workflow](../.github/workflows/build.yaml) builds and publishes the Nix image for amd64 and arm64 on same-repository PRs, pushes to `master`, and release tags. Release tags publish both `<version>-nix` and `latest-nix`. An active Chromium override selects `ubuntu-x64-3xlarge` for amd64 and `ubuntu-arm64-2xlarge` for arm64. [Docker Tests](../.github/workflows/docker-test.yaml) skips Nix acceptance while the override is active to avoid a second Chromium build on each architecture. Default-image acceptance still runs.

## Update Chromium

Renovate owns routine updates to `flake.lock`. Once the pin reaches or exceeds the override version, CI uses the ordinary runners and resumes Nix acceptance tests.

If a Linux stable fix is needed before the pinned Nixpkgs provides it, prepare an override in the renderer checkout. From the repository root on Linux with Nix, Git, and jq, run the updater from the **pinned Nixpkgs revision**. It may download several GiB of Chromium source and dependencies to compute hashes.

```sh
rev=$(jq -r '.nodes.nixpkgs.locked.rev' flake.lock)
work=$(mktemp -d)
(
  set -e
  cd "$work"
  git init -q
  git fetch --quiet --depth=1 https://github.com/NixOS/nixpkgs.git "$rev"
  git checkout --quiet --detach FETCH_HEAD
  recipe=pkgs/applications/networking/browsers/chromium
  jq -r '"Pinned Chromium \(.chromium.version)"' "$recipe/info.json"
  NIX_PATH="nixpkgs=$PWD" "$recipe/update.mjs" --chromium
  jq -S '{chromium}' "$recipe/info.json" > candidate.json
)
jq -r '"Candidate Chromium \(.chromium.version)"' "$work/candidate.json"
```

If the candidate is the newer release you need, copy the complete entry. Force the image derivation to evaluate on each architecture and check its build dependencies for that exact Chromium version. A plain `nix flake check --no-build` can pass even when the image cannot be built.

```sh
cp "$work/candidate.json" nix/chromium-info.json
check_image_chromium() (
  set -eu
  for system in x86_64-linux aarch64-linux; do
    image=$(nix eval --raw --no-update-lock-file ".#packages.$system.image.drvPath")
    printf '%s\n' "$system"
    nix-store --query --requisites "$image" | grep -F -- "-chromium-unwrapped-$1.drv"
  done
)
check_image_chromium "$(jq -er '.chromium.version' nix/chromium-info.json)"
```

When the new major release has no Nixpkgs recipe yet, prepare the packaging changes in this repository.

1. Read the last major-version update to the Chromium package **before the Nixpkgs pin**. Deepen the shallow fetch of the same revision if needed. Use that diff to identify where the recipe depended on the old release, such as a tool version, build flag, or architecture-specific patch.
2. Check each proposed change against Chromium's actual release sources. The updater records the source revision in `.chromium.DEPS.src.rev`. Inspect `DEPS`, build files, and dependency lockfiles at that revision. For an existing patch, check whether its change is already in the release. Derive tool revisions from those files and compute hashes using the recipe's fetchers.
3. Use the first underlying evaluation error to narrow the repair. Put any required recipe edits or patches under `nix/` and wire them into [chromium.nix](chromium.nix) only for the affected override. If evaluation already passes, review the previous transition for build-time assumptions that evaluation cannot exercise. A change to source fetching or unpacking requires recomputing the affected hash.

Make any new recipe files visible to the Git flake with `git add -N <path>`. Repeat the version check after the repair. With `chromium-info.json` temporarily set to `{}`, run `check_image_chromium` with the pinned version printed by the updater, then restore the candidate.

Leave the user a reviewable diff containing the metadata and any required recipe changes. Report the Chromium source evidence for each recipe change, the selected version on both architectures, and anything that still needs compilation to verify. The user reviews and pushes the branch and opens or updates a PR. Check both architecture builds on the PR. If either fails, use the first failing build command and its diagnostic to revise the patch.

When a Renovate pin catches up, selection automatically returns to Nixpkgs. Reset `chromium-info.json` to `{}` and remove any override-specific recipe changes in that PR.

### Packaging Chromium 158 with the 157 recipe

With Nixpkgs pinned at `d5cc6833`, the updater generated metadata for `158.0.8127.36`, but evaluating either image failed on the recipe's missing esbuild entry for major `158`. The 156→157 packaging diff identified the release-specific inputs. Comparing them with Chromium 158's sources yielded the following proposed changes. [156→157 packaging diff](https://github.com/NixOS/nixpkgs/commit/cbe094a83aad0c6d7f9ddcebf2401e724ec62272) · [Pinned recipe](https://github.com/NixOS/nixpkgs/blob/d5cc6833999746cffdfe0598f7bc801a57b887a9/pkgs/applications/networking/browsers/chromium/common.nix#L323)

- **DevTools and esbuild.** Reuse native esbuild `0.28.2` for each architecture. DevTools 158 still imports its matching JS package, but moved `node_modules` from Git into a separate archive. The pinned updater skips non-Git dependencies. Fetch the archive and unpack it directly into `third_party/devtools-frontend/src/node_modules` before patching shebangs and running GN. The archive has no enclosing directory to strip. Its verified flat `fetchurl` hash is `sha256-xn9yYiB3+NIKDWQeY/lXchIFd7hGrOPj1DTbnxMYg1A=`. Place the native executable at `third_party/esbuild/esbuild` inside the DevTools tree, where the wrapper expects it. [DevTools archive declaration](https://chromium.googlesource.com/devtools/devtools-frontend/+/047c90eca37c7170d826d72245d5df9281989cf3/DEPS#262) · [esbuild wrapper](https://chromium.googlesource.com/devtools/devtools-frontend/+/047c90eca37c7170d826d72245d5df9281989cf3/scripts/build/esbuild.js#20)
- **Crubit and the arm64 patch.** Use Chromium's Crubit revision `d757594f6f7ac27c0cf9c8ea63d9136f8e1f77ae` and prefetch its source and Cargo hashes. Its source already contains the old Rust compatibility fix. The arm64 video decoding source likewise already includes `base/logging.h`. Apply those two patches only to the older recipe. Keep the pinned compiler for this attempt and let CI verify Crubit's compatibility with it. [Crubit pin](https://chromium.googlesource.com/chromium/src/+/14d6f648e8cd7ad750a4044200ce94d3321be260/tools/rust/update_rust.py#47) · [Crubit patch result](https://chromium.googlesource.com/external/github.com/google/crubit/+/d757594f6f7ac27c0cf9c8ea63d9136f8e1f77ae/cc_bindings_from_rs/rustversion_impl.rs#45) · [arm64 patch result](https://chromium.googlesource.com/chromium/src/+/14d6f648e8cd7ad750a4044200ce94d3321be260/media/gpu/sandbox/hardware_video_decoding_sandbox_hook_linux.cc#10)
- **Other inputs.** Take GN and depot_tools revisions from the release's `DEPS` and keep the updater's full dependency graph, including new Git dependencies such as `openh264rs` and `ml_drift`. Chromium's own Node lockfile and gnrt's Cargo lockfile were byte-identical across these releases. Keep the existing URL rewrites and verify the corresponding hashes with the same fetchers. The BiDi archive, TypeScript version, and V8's `v8_use_metagen` flag also carry over from 157. [Chromium 158 DEPS](https://chromium.googlesource.com/chromium/src/+/14d6f648e8cd7ad750a4044200ce94d3321be260/DEPS)

## Update the Go builder

Keep the Go builder in [renderer.nix](renderer.nix) aligned with `go.mod`. After Go dependency changes, use `pkgs.lib.fakeHash` for `vendorHash`, build once, and replace it with the hash reported by Nix.
