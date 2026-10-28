#!/usr/bin/env bash
# Clears only caches that rebuild automatically. Never touches go/pkg/mod, Docker/Colima, or project directories.
# Usage: ./mac-cleanup.sh            run all steps
#        ./mac-cleanup.sh go gradle  run only the named steps (go gradle js brew)
set -u

VOL=/System/Volumes/Data
avail() { df -k "$VOL" | awk 'NR==2{print $4}'; }
has() { command -v "$1" >/dev/null 2>&1; }

step_go() {
  has go && go clean -cache
  has golangci-lint && golangci-lint cache clean
  rm -rf ~/Library/Caches/goimports ~/Library/Caches/gopls ~/Library/Caches/golangci-lint
}

# Keep ~/.gradle/gradle.properties and init.d: they may hold private repository credentials.
step_gradle() {
  pkill -f GradleDaemon 2>/dev/null
  rm -rf ~/.gradle/caches ~/.gradle/daemon ~/.gradle/wrapper/dists
}

step_js() {
  has yarn && yarn cache clean
  has npm && npm cache clean --force
  has pnpm && pnpm store prune
  has pip3 && pip3 cache purge
}

step_brew() {
  has brew || return 0
  brew cleanup --prune=all
  brew autoremove
}

steps=("$@")
[ ${#steps[@]} -eq 0 ] && steps=(go gradle js brew)

start=$(avail)
for s in "${steps[@]}"; do
  before=$(avail)
  echo "==> $s"
  "step_$s"
  echo "    freed $(( ($(avail) - before) / 1024 )) MB"
done
echo "total freed $(( ($(avail) - start) / 1024 / 1024 )) GB, free now: $(df -h "$VOL" | awk 'NR==2{print $4}')"

bash scripts/mac-cleanup.sh